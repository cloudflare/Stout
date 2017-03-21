package amazonprovider

import (
	"errors"
	"fmt"

	"github.com/eagerio/Stout/src/providers/amazonprovider/dns"
	"github.com/urfave/cli"
)

// Create a new user, CloudFront distrbution, s3 bucket, route53 route
func (a *client) CreateDNS(c cli.Context) error {
	err := checkForAWS()
	if err != nil {
		return err
	}

	awsKey := ""
	awsSecret := ""

	domain := ""
	cdnDomainName := ""

	// if s3Session == nil {
	// 	s3Session = openS3(awsKey, awsSecret, awsRegion)
	// }
	// if iamSession == nil {
	// 	iamSession = openIAM(awsKey, awsSecret, awsRegion)
	// }
	if r53Session == nil {
		r53Session = openRoute53(awsKey, awsSecret)
	}
	// if cfSession == nil {
	// 	cfSession = openCloudFront(awsKey, awsSecret)
	// }

	//official sdk connection
	// if awsSession == nil {
	// 	awsSession = session.New(&aws.Config{
	// 		Region:      aws.String(awsRegion),
	// 		Credentials: credentials.NewStaticCredentials(awsKey, awsSecret, ""),
	// 	})
	// }

	// fmt.Println("Checking for available SSL/TLS certificates")
	// certificateARN, err := setUpSSL(awsSession, domain, createSSL, noSSL)
	// if err != nil {
	// 	return errors.New("Error while processing SSL/TLS certificates\n" + err.Error())
	// }
	//
	// if certificateARN == "" {
	// 	fmt.Println("Will set up CloudFront distrbution without SSL/TLS")
	// }

	// fmt.Println("Creating Bucket")
	// err = CreateBucket(domain)
	// if err != nil {
	// 	return errors.New("Error creating S3 bucket\n" + err.Error())
	// }

	// fmt.Println("Loading/Creating CloudFront Distribution")
	// cdnDomainName, err := GetDistribution(awsSession, certificateARN, createSSL, domain, awsRegion)
	// if err != nil {
	// 	return errors.New("Error loading/creating CloudFront distribution\n" + err.Error())
	// }

	fmt.Println("Adding Route")
	err = dns.UpdateR53Route(r53Session, domain, cdnDomainName)
	if err != nil {
		return errors.New("Error adding route to Route53 DNS config\n" + err.Error())
	}

	// 	if !noUser {
	// 		key, err := CreateS3User(domain)
	// 		if err != nil {
	// 			return errors.New("Error creating user\n" + err.Error())
	// 		}
	//
	// 		fmt.Println("An access key has been created with just the permissions required to deploy / rollback this site")
	// 		fmt.Println("It is strongly recommended you use this limited account to deploy this project in the future")
	// 		fmt.Println()
	// 		fmt.Printf("ACCESS_KEY_ID=%s\n", key.Id)
	// 		fmt.Printf("ACCESS_KEY_SECRET=%s\n\n", key.Secret)
	//
	// 		if terminal.IsTerminal(int(os.Stdin.Fd())) {
	// 			fmt.Println(`You can either add these credentials to the deploy.yaml file,
	// or specify them as arguments to the stout deploy / stout rollback commands.
	// You MUST NOT add them to the deploy.yaml file if this project is public
	// (i.e. a public GitHub repo).
	//
	// If you can't add them to the deploy.yaml file, you can specify them as
	// arguments on the command line.  If you use a build system like CircleCI, you
	// can add them as environment variables and pass those variables to the deploy
	// commands (see the README).
	//
	// Your first deploy command might be:
	//
	// 	stout deploy --domain ` + domain + ` --key ` + key.Id + ` --secret '` + key.Secret + `'
	// `)
	// 		}
	//
	// 	}

	return nil
}
