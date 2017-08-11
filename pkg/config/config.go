package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Imap struct {
		Server   string
		Username string
		Password string
	}
	GitHub struct {
		Username string
	}
	CIRobot struct {
		Enabled bool
	}
	MergeRobot struct {
		Enabled bool
	}
	RobotCommands struct {
		Enabled bool
	}
	OpenshiftBot struct {
		Enabled bool
	}
	MoveTo string
}

func Read(path string) (*Config, error) {
	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			b, _ := yaml.Marshal(&Config{})
			ioutil.WriteFile(path, b, 0600)
			return nil, fmt.Errorf("%s created: please edit it and re-run", path)
		}

		return nil, err
	}

	// config file contains password: ensure group and other permissions are not
	// set
	if st.Mode().Perm()&077 != 0 {
		err = os.Chmod(path, st.Mode().Perm()&^077)
		if err != nil {
			return nil, err
		}
	}

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config *Config
	err = yaml.Unmarshal(b, &config)
	if err != nil {
		return nil, err
	}

	err = validate(config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func validate(config *Config) error {
	var errors []error

	if config.Imap.Server == "" {
		errors = append(errors, fmt.Errorf("imap.server is empty"))
	}
	if config.Imap.Username == "" {
		errors = append(errors, fmt.Errorf("imap.username is empty"))
	}
	if config.Imap.Password == "" {
		errors = append(errors, fmt.Errorf("imap.password is empty"))
	}
	if config.MoveTo == "" {
		errors = append(errors, fmt.Errorf("moveto is empty"))
	}
	if (config.CIRobot.Enabled || config.MergeRobot.Enabled || config.RobotCommands.Enabled) && config.GitHub.Username == "" {
		errors = append(errors, fmt.Errorf("github.username is empty"))
	}

	if errors != nil {
		errorStrings := make([]string, 0, len(errors))
		for _, err := range errors {
			errorStrings = append(errorStrings, err.Error())
		}
		return fmt.Errorf(strings.Join(errorStrings, "\n"))
	}

	return nil
}
