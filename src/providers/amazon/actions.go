package amazon

import (
	"errors"
	"fmt"

	"github.com/cloudflare/stout/src/providers/amazon/cdn"
	"github.com/cloudflare/stout/src/providers/amazon/dns"
	"github.com/cloudflare/stout/src/providers/amazon/fs"
	"github.com/cloudflare/stout/src/types"
)

func (a *client) CreateDNS(g types.GlobalFlags, c types.CreateFlags, cdnDomainName string) error {
	fmt.Println("Adding Route")
	err := dns.CreateR53Route(r53Session, g.Domain, cdnDomainName)
	if err != nil {
		return errors.New("Error adding route to Route53 DNS config\n" + err.Error())
	}

	return nil
}

func (a *client) CreateFS(g types.GlobalFlags, c types.CreateFlags) (string, error) {
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

func (a *client) CreateCDN(g types.GlobalFlags, c types.CreateFlags, fsDomain string) (string, error) {
	fmt.Println("Checking for available SSL/TLS certificates")
	certificateARN, err := setUpSSL(awsSession, g.Domain, a.CreateSSL)
	if err != nil {
		return "", errors.New("Error while processing SSL/TLS certificates\n" + err.Error())
	}

	if certificateARN == "" {
		fmt.Println("Will set up CloudFront distribution without SSL/TLS")
	}

	fmt.Println("Loading/Creating CloudFront Distribution")
	cdnDomainName, err := cdn.GetCFDistribution(cfSession, certificateARN, a.CreateSSL, g.Domain, a.Region)
	if err != nil {
		return "", errors.New("Error loading/creating CloudFront distribution\n" + err.Error())
	}

	return cdnDomainName, nil
}

func (a *client) FSProviderFuncs(g types.GlobalFlags) (types.FSProviderFunctions, error) {
	return fs.FSProviderFuncs(s3Session, g.Domain)
}

func (a *client) AddVerificationRecord(g types.GlobalFlags, c types.CreateFlags, recordType types.DNSRecordType, name string, value string) error {
	return dns.AddVerificationRecord(r53Session, g.Domain, string(recordType), name, value)
}
