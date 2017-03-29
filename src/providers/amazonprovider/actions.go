package amazonprovider

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/eagerio/Stout/src/providers"
	"github.com/eagerio/Stout/src/providers/amazonprovider/cdn"
	"github.com/eagerio/Stout/src/providers/amazonprovider/dns"
	"github.com/eagerio/Stout/src/providers/amazonprovider/fs"
)

// Create a route53 route
func (a *client) CreateDNS(g providers.GlobalFlags, c providers.CreateFlags, cdnDomainName string) error {
	fmt.Println("Adding Route")
	err := dns.UpdateR53Route(r53Session, g.Domain, cdnDomainName)
	if err != nil {
		return errors.New("Error adding route to Route53 DNS config\n" + err.Error())
	}

	return nil
}

// Create a new s3 bucket, optionally create a new user
func (a *client) CreateFS(g providers.GlobalFlags, c providers.CreateFlags) error {
	fmt.Println("Creating Bucket")
	err := fs.CreateS3Bucket(s3Session, g.Domain)
	if err != nil {
		return errors.New("Error creating S3 bucket\n" + err.Error())
	}

	if a.AWSNewUser {
		key, err := fs.CreateS3User(iamSession, g.Domain)
		if err != nil {
			return errors.New("Error creating user\n" + err.Error())
		}

		fmt.Println("An access key has been created with just the permissions required to deploy / rollback this site")
		fmt.Println("It is strongly recommended you use this limited account to deploy this project in the future")
		fmt.Println()
		fmt.Printf("ACCESS_KEY_ID=%s\n", key.Id)
		fmt.Printf("ACCESS_KEY_SECRET=%s\n\n", key.Secret)

		if terminal.IsTerminal(int(os.Stdin.Fd())) {
			fmt.Println(`You can either add these credentials to the deploy.yaml file,
	or specify them as arguments to the stout deploy / stout rollback commands.
	You MUST NOT add them to the deploy.yaml file if this project is public
	(i.e. a public GitHub repo).

	If you can't add them to the deploy.yaml file, you can specify them as
	arguments on the command line.  If you use a build system like CircleCI, you
	can add them as environment variables and pass those variables to the deploy
	commands (see the README).

	Your first deploy command might be:

		stout deploy --domain ` + g.Domain + ` --key ` + key.Id + ` --secret '` + key.Secret + `'
	`)
		}

	}

	return nil
}

// Create a new CloudFront distrbution
func (a *client) CreateCDN(g providers.GlobalFlags, c providers.CreateFlags) (string, error) {
	fmt.Println("Checking for available SSL/TLS certificates")
	certificateARN, err := setUpSSL(awsSession, g.Domain, c.CreateSSL, c.NoSSL)
	if err != nil {
		return "", errors.New("Error while processing SSL/TLS certificates\n" + err.Error())
	}

	if certificateARN == "" {
		fmt.Println("Will set up CloudFront distrbution without SSL/TLS")
	}

	fmt.Println("Loading/Creating CloudFront Distribution")
	cdnDomainName, err := cdn.GetCFDistribution(awsSession, certificateARN, c.CreateSSL, g.Domain, a.AWSRegion)
	if err != nil {
		return "", errors.New("Error loading/creating CloudFront distribution\n" + err.Error())
	}

	return cdnDomainName, nil
}

// Deploy a new version
func (a *client) DeployFS(g providers.GlobalFlags, d providers.DeployFlags) error {
	return fs.Deploy(s3Session, g.Domain, d.Root, d.Files, d.Dest)
}

// Deploy a new version
func (a *client) RollbackFS(g providers.GlobalFlags, r providers.RollbackFlags) error {
	return fs.Rollback(s3Session, g.Domain, r.Dest, r.Version)
}
