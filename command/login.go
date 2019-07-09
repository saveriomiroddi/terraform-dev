package command

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform/svchost"
	"github.com/hashicorp/terraform/tfdiags"
)

// LoginCommand is a Command implementation that runs an interactive login
// flow for a remote service host. It then stashes credentials in a tfrc
// file in the user's home directory.
type LoginCommand struct {
	Meta
}

// Run implements cli.Command.
func (c *LoginCommand) Run(args []string) int {
	args, err := c.Meta.process(args, false)
	if err != nil {
		return 1
	}

	cmdFlags := c.Meta.defaultFlagSet("login")
	var intoFile string
	cmdFlags.StringVar(&intoFile, "into-file", "", "set the file that the credentials will be appended to")
	cmdFlags.Usage = func() { c.Ui.Error(c.Help()) }
	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	args = cmdFlags.Args()
	if len(args) > 1 {
		c.Ui.Error(
			"The login command expects at most one argument: the host to log in to.")
		cmdFlags.Usage()
		return 1
	}

	var diags tfdiags.Diagnostics

	givenHostname := "app.terraform.io"
	if len(args) != 0 {
		givenHostname = args[0]
	}

	dispHostname := svchost.ForDisplay(givenHostname)
	hostname, err := svchost.ForComparison(givenHostname)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Invalid hostname",
			fmt.Sprintf("The given hostname %q is not valid: %s.", givenHostname, err.Error()),
		))
	}

	// From now on, since we've validated the given hostname, we should use
	// dispHostname in the UI to ensure we're presenting it in the canonical
	// form, in case that helpers users with debugging when things aren't
	// working.

	host, err := c.Services.Discover(hostname)
	if err != nil {
		diags = diags.Append(tfdiags.Sourceless(
			tfdiags.Error,
			"Service discovery failed for"+dispHostname,

			// Contrary to usual Go idiom, the Discover function returns
			// full sentences with initial capitalization in its error messages,
			// and they are written with the end-user as the audience. We
			// only need to add the trailing period to make them consistent
			// with our usual error reporting standards.
			err.Error()+".",
		))
	}

	fmt.Printf("Host is %#v\n", host)

	return 0
}

// Help implements cli.Command.
func (c *LoginCommand) Help() string {
	defaultFile := c.defaultOutputFile()
	if defaultFile == "" {
		// Because this is just for the help message and it's very unlikely
		// that a user wouldn't have a functioning home directory anyway,
		// we'll just use a placeholder here. The real command has some
		// more complex behavior for this case. This result is not correct
		// on all platforms, but given how unlikely we are to hit this case
		// that seems okay.
		defaultFile = "~/.terraform/credentials.tfrc"
	}

	helpText := fmt.Sprintf(`
Usage: terraform login [options] [hostname]

  Retrieves an authentication token for the given hostname, if it supports
  automatic login, and saves it in a credentials file in your home directory.

  If no hostname is provided, the default hostname is app.terraform.io, to
  log in to Terraform Cloud.

  If not overridden by the -into-file option, the output file is:
      %s

Options:

  -into-file=....     Override which file the credentials block will be written
                      to. If this file already exists then it must have valid
                      HCL syntax and Terraform will update it in-place.
`, defaultFile)
	return strings.TrimSpace(helpText)
}

// Synopsis implements cli.Command.
func (c *LoginCommand) Synopsis() string {
	return "Obtain and save credentials for a remote host"
}

func (c *LoginCommand) defaultOutputFile() string {
	if c.CLIConfigDir == "" {
		return "" // no default available
	}
	return filepath.Join(c.CLIConfigDir, "credentials.tfrc")
}
