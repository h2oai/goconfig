// Package goconfig uses a struct as input and populates the
// fields of this struct with parameters fom command
// line, environment variables and configuration file.
package goconfig

import (
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/h2oai/goconfig/goenv"
	"github.com/h2oai/goconfig/goflags"
	"github.com/h2oai/goconfig/structtag"
	"github.com/h2oai/goconfig/validate"
)

// Fileformat struct holds the functions to Load the file containing the settings
type Fileformat struct {
	Extension   string
	Load        func(config interface{}) (err error)
	PrepareHelp func(config interface{}) (help string, err error)
}

var (

	// Tag to set main name of field
	Tag = "cfg"

	// TagDefault to set default value
	TagDefault = "cfgDefault"

	// TagHelper to set usage help line
	TagHelper = "cfgHelper"

	// Path sets default config path
	Path string

	// File name of default config file
	File string

	// FileRequired config file required
	FileRequired bool

	// HelpString temporarily saves help
	HelpString string

	// PrefixFlag is a string that would be placed at the beginning of the generated Flag tags.
	PrefixFlag string

	// PrefixEnv is a string that would be placed at the beginning of the generated Event tags.
	PrefixEnv string

	// ErrFileFormatNotDefined Is the error that is returned when there is no defined configuration file format.
	ErrFileFormatNotDefined = errors.New("file format not defined")

	//Usage is a function to show the help, can be replaced by your own version.
	Usage func()

	// Formats is the list of registered formats.
	Formats []Fileformat

	// FileEnv is the enviroment variable that define the config file
	FileEnv string

	// PathEnv is the enviroment variable that define the config file path
	PathEnv string

	// WatchConfigFile is the flag to update the config when the config file changes
	WatchConfigFile bool

	// DisableFlags on the command line
	DisableFlags bool

	// Convert kebabcase (dashes) cmd args to snakecase (underscores) environment variables
	KebabCfgToSnakeEnv bool
)

func findFileFormat(extension string) (format Fileformat, err error) {
	format = Fileformat{}
	for _, f := range Formats {
		if f.Extension == extension {
			format = f
			return
		}
	}
	err = ErrFileFormatNotDefined
	return
}

func init() {
	Usage = DefaultUsage
	Path = "./"
	File = ""
	FileRequired = false

	FileEnv = "GO_CONFIG_FILE"
	PathEnv = "GO_CONFIG_PATH"

	WatchConfigFile = false
}

// Parse configuration
func Parse(config interface{}) (err error) {
	goenv.Prefix = PrefixEnv
	goenv.Setup(Tag, TagDefault, KebabCfgToSnakeEnv)
	err = structtag.SetBoolDefaults(config, "")
	if err != nil {
		return
	}

	lookupEnv()

	ext := path.Ext(File)
	if ext != "" {
		if err = loadConfigFromFile(ext, config); err != nil {
			return
		}
	}

	goenv.Prefix = PrefixEnv
	goenv.Setup(Tag, TagDefault, KebabCfgToSnakeEnv)
	err = goenv.Parse(config)
	if err != nil {
		return
	}

	if !DisableFlags {
		goflags.Prefix = PrefixFlag
		goflags.Setup(Tag, TagDefault, TagHelper)
		goflags.Usage = Usage
		goflags.Preserve = true
		err = goflags.Parse(config)
		if err != nil {
			return
		}
	}

	validate.Prefix = PrefixFlag
	validate.Setup(Tag, TagDefault)
	err = validate.Parse(config)

	return
}

// PrintDefaults print the default help
func PrintDefaults() {
	if File != "" {
		fmt.Printf("Config file %q:\n", filepath.Join(Path, File))
		fmt.Println(HelpString)
	}
}

// DefaultUsage is assigned for Usage function by default
func DefaultUsage() {
	fmt.Println("Usage")
	goflags.PrintDefaults()
	goenv.PrintDefaults()
	PrintDefaults()
}

func lookupEnv() {
	pref := PrefixEnv
	if pref != "" {
		pref = pref + structtag.TagSeparator
	}

	if val, set := os.LookupEnv(pref + FileEnv); set {
		File = val
	}

	if val, set := os.LookupEnv(pref + PathEnv); set {
		Path = val
	}
}

func loadConfigFromFile(ext string, config interface{}) (err error) {
	var format Fileformat
	format, err = findFileFormat(ext)
	if err != nil {
		return
	}
	err = format.Load(config)
	if err != nil {
		return
	}
	HelpString, err = format.PrepareHelp(config)
	if err != nil {
		return
	}

	return
}

func asyncParse(config interface{}, w *fsnotify.Watcher, chErr chan<- error, chUp chan<- int64) {
	var state uint
	for {
		select {
		case ev := <-w.Events:
			// these event check are needed for vi-like editors that uses a swap file when saving
			// other editors like nano directly writes to the file
			if ev.Op&fsnotify.Rename == fsnotify.Rename && (state == 0) {
				state |= (1 << 0)
			} else if ev.Op&fsnotify.Chmod == fsnotify.Chmod && (state == 1) {
				state |= (1 << 1)
			} else if ev.Op&fsnotify.Remove == fsnotify.Remove && (state == 3) {
				state |= (1 << 2)
			}

			if (ev.Op&fsnotify.Write == fsnotify.Write) || (state == 7) {
				if err := loadConfigFromFile(path.Ext(File), config); err != nil {
					chErr <- err
					break
				}

				chUp <- time.Now().Unix()

				state = 0
				w.Add(path.Join(Path, File))
			}

		case err := <-w.Errors:
			chErr <- err
			break
		}
	}
}

// ParseAndWatch configuration returns a channel for errors while watching files
// and anorther when each update has been detected
func ParseAndWatch(config interface{}) (chChanges chan int64, chErr chan error, err error) {
	chErr = make(chan error, 1)
	chChanges = make(chan int64, 1)

	lookupEnv()

	ext := path.Ext(File)
	if ext != "" {
		if err = loadConfigFromFile(ext, config); err != nil {
			return
		}

		if WatchConfigFile {
			watcher, err := fsnotify.NewWatcher()
			if err != nil {
				return chChanges, chErr, err
			}
			if err = watcher.Add(path.Join(Path, File)); err != nil {
				return chChanges, chErr, err
			}
			go asyncParse(config, watcher, chErr, chChanges)
		}
	}

	validate.Prefix = PrefixFlag
	validate.Setup(Tag, TagDefault)
	err = validate.Parse(config)

	return
}
