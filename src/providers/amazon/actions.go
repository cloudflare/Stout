package amazon

import (
	"errors"
	"fmt"

	"github.com/eagerio/Stout/src/providers"
	"github.com/eagerio/Stout/src/providers/amazon/cdn"
	"github.com/eagerio/Stout/src/providers/amazon/dns"
	"github.com/eagerio/Stout/src/providers/amazon/fs"
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
func (a *client) CreateFS(g providers.GlobalFlags, c providers.CreateFlags) (string, error) {
	fmt.Println("Getting/Creating S3 Bucket")
	fsDomain, err := fs.CreateS3Bucket(s3Session, g.Domain, a.Region)
	if err != nil {
		return "", errors.New("Error creating S3 bucket\n" + err.Error())
	}

	if a.NewUser {
		key, err := fs.CreateS3User(iamSession, g.Domain)
		if err != nil {
			return "", errors.New("Error creating user\n" + err.Error())
		}

		fmt.Println("An access key has been created with just the permissions required to deploy / rollback this site")
		fmt.Println("It is strongly recommended you use this limited account to deploy this project in the future")
		fmt.Println()
		fmt.Printf("key=%s\n", *key.AccessKeyId)
		fmt.Printf("secret=%s\n\n", *key.SecretAccessKey)
	}

	return fsDomain, nil
}

// Create a new CloudFront distrbution
func (a *client) CreateCDN(g providers.GlobalFlags, c providers.CreateFlags, fsDomain string) (string, error) {
	fmt.Println("Checking for available SSL/TLS certificates")
	certificateARN, err := setUpSSL(awsSession, g.Domain, a.CreateSSL)
	if err != nil {
		return "", errors.New("Error while processing SSL/TLS certificates\n" + err.Error())
	}

	if certificateARN == "" {
		fmt.Println("Will set up CloudFront distrbution without SSL/TLS")
	}

	fmt.Println("Loading/Creating CloudFront Distribution")
	cdnDomainName, err := cdn.GetCFDistribution(cfSession, certificateARN, a.CreateSSL, g.Domain, a.Region)
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
