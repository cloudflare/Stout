package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"code.google.com/p/go.net/publicsuffix"
	"github.com/zackbloom/goamz/cloudfront"
	"github.com/zackbloom/goamz/iam"
	"github.com/zackbloom/goamz/route53"
	"github.com/zackbloom/goamz/s3"
	"golang.org/x/crypto/ssh/terminal"
)

func CreateBucket(options Options) error {
	bucket := s3Session.Bucket(options.Bucket)

	err := bucket.PutBucket("public-read")
	// TODO Ignore found error
	if err != nil {
		return err
	}

	err = bucket.PutBucketWebsite(s3.WebsiteConfiguration{
		IndexDocument: &s3.IndexDocument{"index.html"},
		ErrorDocument: &s3.ErrorDocument{"error.html"},
	})
	if err != nil {
		return err
	}

	err = bucket.PutPolicy([]byte(`
		{
			"Version": "2008-10-17",
			"Statement": [
				{
					"Sid": "PublicReadForGetBucketObjects",
					"Effect": "Allow",
					"Principal": {
						"AWS": "*"
					},
					"Action": "s3:GetObject",
					"Resource": "arn:aws:s3:::` + options.Bucket + `/*"
				}
			]
		}`,
	))
	if err != nil {
		return err
	}

	return nil
}

func GetDistribution(options Options) (dist cloudfront.DistributionSummary, err error) {
	distP, err := cfSession.FindDistributionByAlias(options.Bucket)
	if err != nil {
		return
	}

	if distP != nil {
		log.Println("CloudFront distribution found with the provided bucket name, assuming config matches.")
		log.Println("If you run into issues, delete the distribution and rerun this command.")

		dist = *distP
		return
	}

	conf := cloudfront.DistributionConfig{
		Origins: cloudfront.Origins{
			cloudfront.Origin{
				Id:             "S3-" + options.Bucket,
				DomainName:     options.Bucket + ".s3-website-" + options.AWSRegion + ".amazonaws.com",
				S3OriginConfig: &cloudfront.S3OriginConfig{},
			},
		},
		DefaultRootObject: "index.html",
		PriceClass:        "PriceClass_All",
		Enabled:           true,
		DefaultCacheBehavior: cloudfront.CacheBehavior{
			TargetOriginId:       "S3-" + options.Bucket,
			ViewerProtocolPolicy: "allow-all",
			AllowedMethods: cloudfront.AllowedMethods{
				Allowed: []string{"GET", "HEAD"},
				Cached:  []string{"GET", "HEAD"},
			},
		},
		ViewerCertificate: &cloudfront.ViewerCertificate{
			CloudFrontDefaultCertificate: true,
		},
		Aliases: cloudfront.Aliases{
			options.Bucket,
		},
	}

	return cfSession.Create(conf)
}

func CreateUser(options Options) (key iam.AccessKey, err error) {
	name := options.Bucket + "_deploy"

	_, err = iamSession.CreateUser(name, "/")
	if err != nil {
		return
	}

	_, err = iamSession.PutUserPolicy(name, name, `
		{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Action": [
						"s3:DeleteObject",
						"s3:ListBucket",
						"s3:PutObject",
						"s3:PutObjectAcl",
						"s3:GetObject"
					],
					"Resource": [
						"arn:aws:s3:::`+options.Bucket+`", "arn:aws:s3:::`+options.Bucket+`/*"
					]
				}
			]
		}`,
	)
	if err != nil {
		return
	}

	keyResp, err := iamSession.CreateAccessKey(name)
	if err != nil {
		return
	}

	return keyResp.AccessKey, nil
}

func UpdateRoute(options Options, dist cloudfront.DistributionSummary) error {
	zoneName, err := publicsuffix.EffectiveTLDPlusOne(options.Bucket)
	if err != nil {
		return err
	}

	resp, err := r53Session.ListHostedZonesByName(zoneName, "", 100)

	if resp.IsTruncated {
		panic("More than 100 zones in the account")
	}

	// TODO: Figure out what happens when the zone isnt found
	noZone := false
	// END

	if noZone {
		fmt.Printf("A Route 53 hosted zone was not found for %s", zoneName)
		if zoneName != options.Bucket {
			fmt.Println("If you would like to use Route 53 to manage your DNS, create a zone for this domain, and update your registrar's configuration to point to the DNS servers Amazon provides and rerun this command.  Note that you must copy any existing DNS configuration you have to Route 53 if you do not wish existing services hosted on this domain to stop working.")
			fmt.Printf("If you would like to continue to use your existing DNS, create a CNAME record pointing %s to %s and the site setup will be finished.", options.Bucket, dist.DomainName)
		} else {
			fmt.Println("Since you are hosting the root of your domain, using an alternative DNS host is unfortunately not possible.")
			fmt.Println("If you wish to host your site at the root of your domain, you must switch your sites DNS to Amazon's Route 53 and retry this command.")
		}
	}

	if err != nil {
		return err
	}

	if len(resp.HostedZones) > 1 {
		panic("Multiple matching hosted zones found")
	}
	if len(resp.HostedZones) == 0 {
		panic("Hosted zone not listed")
	}

	zone := resp.HostedZones[0]

	fmt.Printf("Adding %s to %s Route 53 zone\n", options.Bucket, zoneName)
	_, err = r53Session.ChangeResourceRecordSet(&route53.ChangeResourceRecordSetsRequest{
		Changes: []route53.Change{
			route53.Change{
				Action: "CREATE",
				Name:   options.Bucket,
				TTL:    3600,
				AliasTarget: route53.AliasTarget{
					HostedZoneId:         dist.Id,
					DNSName:              dist.DomainName,
					EvaluateTargetHealth: false,
				},
			},
		},
	}, zone.Id)

	if err != nil {
		return err
	}

	return nil
}

func Create(options Options) {
	if s3Session == nil {
		s3Session = openS3(options.AWSKey, options.AWSSecret, options.AWSRegion)
	}
	if iamSession == nil {
		iamSession = openIAM(options.AWSKey, options.AWSSecret, options.AWSRegion)
	}
	if r53Session == nil {
		r53Session = openRoute53(options.AWSKey, options.AWSSecret)
	}
	if cfSession == nil {
		cfSession = openCloudFront(options.AWSKey, options.AWSSecret)
	}

	_, err := exec.LookPath("aws")
	if err != nil {
		fmt.Println("The aws CLI executable was not found in the PATH")
		fmt.Println("Install it from http://aws.amazon.com/cli/ and try again")
	}

	log.Println("Creating Bucket")
	err = CreateBucket(options)

	if err != nil {
		fmt.Println("Error creating S3 bucket")
		fmt.Println(err)
		return
	}

	log.Println("Loading/Creating CloudFront Distribution")
	dist, err := GetDistribution(options)

	if err != nil {
		fmt.Println("Error loading/creating CloudFront distribution")
		fmt.Println(err)
		return
	}

	log.Println("Adding Route")
	err = UpdateRoute(options, dist)

	if err != nil {
		fmt.Println("Error adding route to Route53 DNS config")
		fmt.Println(err)
		return
	}

	key, err := CreateUser(options)

	fmt.Println("An access key has been created with just the permissions required to deploy / rollback this site")
	fmt.Println("It is strongly recommended you use this limited account to deploy this project in the future")
	fmt.Printf("ACCESS_KEY_ID=%s\n", key.Id)
	fmt.Printf("ACCESS_KEY_SECRET=%s\n", key.Secret)

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
		  
		  stout deploy --bucket ` + options.Bucket + ` --key ` + key.Id + ` --secret '` + key.Secret + `'

		`)
	}

	fmt.Printf("Site %s created", options.Bucket)
	fmt.Println("You can begin deploying now (stout deploy), but it can take up to ten minutes for your site to begin to work")
	fmt.Println("Depending on the configuration of your site, you might need to set the 'root', 'dest' or 'files' options to get your deploys working as you wish.  See the README for details.")
	fmt.Println("It's also a good idea to look into the 'env' option, as in real-world situations it usually makes sense to have a development and/or staging site for each of your production sites.")
}

func createCmd() {
	options, _ := parseOptions()
	loadConfigFile(&options)

	if options.Bucket == "" {
		panic("You must specify a bucket")
	}

	if options.AWSKey == "" || options.AWSSecret == "" {
		panic("You must specify your AWS credentials")
	}

	Create(options)
}
