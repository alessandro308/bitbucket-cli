package factory

import (
	"github.com/alessandro308/bitbucket-cli/internal/config"
	"github.com/alessandro308/bitbucket-cli/pkg/browser"
	"github.com/alessandro308/bitbucket-cli/pkg/cmdutil"
	"github.com/alessandro308/bitbucket-cli/pkg/iostreams"
	"github.com/alessandro308/bitbucket-cli/pkg/pager"
	"github.com/alessandro308/bitbucket-cli/pkg/progress"
	"github.com/alessandro308/bitbucket-cli/pkg/prompter"
)

// New constructs a command factory following gh/jk idioms.
func New(appVersion string) (*cmdutil.Factory, error) {
	ios := iostreams.System()

	f := &cmdutil.Factory{
		AppVersion:     appVersion,
		ExecutableName: "bkt",
		IOStreams:      ios,
	}

	f.Browser = browser.NewSystem()
	f.Pager = pager.NewSystem(ios)
	f.Prompter = prompter.New(ios)
	f.Spinner = progress.NewSpinner(ios)

	f.Config = func() (*config.Config, error) {
		return config.Load()
	}

	return f, nil
}
