package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	amazonaws "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acm"
	amazoncloudfront "github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/zackbloom/goamz/iam"
	"github.com/zackbloom/goamz/route53"
	"github.com/zackbloom/goamz/s3"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/net/publicsuffix"
)

/*
* Create bucket with public read policy
* set website index and error documents to index.html and error.html
 */
func CreateBucket(options Options) error {
	bucket := s3Session.Bucket(options.Domain)

	err := bucket.PutBucket("public-read")
	if err != nil {
		return err
	}

	err = bucket.PutBucketWebsite(s3.WebsiteConfiguration{
		IndexDocument: &s3.IndexDocument{
			Suffix: "index.html",
		},
		ErrorDocument: &s3.ErrorDocument{
			Key: "index.html",
		},
	})
	if err != nil {
		return err
	}

	err = bucket.PutPolicy([]byte(`{
			"Version": "2008-10-17",
			"Statement": [
				{
					"Sid": "PublicReadForGetBucketObjects",
					"Effect": "Allow",
					"Principal": {
						"AWS": "*"
					},
					"Action": "s3:GetObject",
					"Resource": "arn:aws:s3:::` + options.Domain + `/*"
				}
			]
		}`,
	))
	if err != nil {
		return err
	}

	return nil
}

/*
* Get cloudfront distribution
* create one if none already exists for the specified domain name in options
* Check if SSL is set up for the distribution and set that up if a ACM Cert ARN is passed in
* Returns distribution domain name
 */
func GetDistribution(amazonSession *session.Session, options Options, certificateARN string) (string, error) {
	cloudfrontService := amazoncloudfront.New(amazonSession)
	dists, err := cloudfrontService.ListDistributions(&amazoncloudfront.ListDistributionsInput{
		MaxItems: amazonaws.Int64(500),
	})
	if err != nil {
		return "", err
	}

	if *dists.DistributionList.IsTruncated {
		return "", errors.New("You have more than 500 distributions, please consider opening a GitHub issue if you require support for this.")
	}

	var currentDist *amazoncloudfront.DistributionSummary
	//check for already existing distributions
	for _, distSummary := range dists.DistributionList.Items {
		for _, alias := range distSummary.Aliases.Items {
			if *alias == options.Domain {
				//matching distribution found
				fmt.Println("CloudFront distribution found with the provided bucket name, assuming config matches.")
				fmt.Println("If you run into issues, delete the distribution and rerun this command.")
				currentDist = distSummary
			}
		}
	}

	if currentDist != nil && !options.CreateSSL {
		// if there is a distribution already and there is no upgrade request, return it
		return *currentDist.DomainName, nil
	} else if currentDist != nil && options.CreateSSL {
		// if there is a distribution and createssl is an option, check if the distribution has a certificate
		distDetail, err := cloudfrontService.GetDistribution(&amazoncloudfront.GetDistributionInput{
			Id: amazonaws.String(*currentDist.Id),
		})
		if err != nil {
			return "", err
		}
		// if there is no certificate installed, update it
		if *distDetail.Distribution.DistributionConfig.ViewerCertificate.CloudFrontDefaultCertificate {
			fmt.Println("Updating current CloudFront distribution with your new certificate")
			distDetail.Distribution.DistributionConfig.ViewerCertificate = &amazoncloudfront.ViewerCertificate{
				ACMCertificateArn:      amazonaws.String(certificateARN),
				Certificate:            amazonaws.String(certificateARN),
				CertificateSource:      amazonaws.String("acm"),
				MinimumProtocolVersion: amazonaws.String("TLSv1"),
				SSLSupportMethod:       amazonaws.String("sni-only"),
			}
			_, err := cloudfrontService.UpdateDistribution(&amazoncloudfront.UpdateDistributionInput{
				DistributionConfig: distDetail.Distribution.DistributionConfig,
				Id:                 distDetail.Distribution.Id,
				IfMatch:            distDetail.ETag,
			})
			if err != nil {
				return "", err
			}
		}
		return *currentDist.DomainName, nil
	}

	//no matching distribution, create one
	var viewerCertificate amazoncloudfront.ViewerCertificate
	if certificateARN != "" {
		viewerCertificate = amazoncloudfront.ViewerCertificate{
			ACMCertificateArn:      amazonaws.String(certificateARN),
			Certificate:            amazonaws.String(certificateARN),
			CertificateSource:      amazonaws.String("acm"),
			MinimumProtocolVersion: amazonaws.String("TLSv1"),
			SSLSupportMethod:       amazonaws.String("sni-only"),
		}
	} else {
		viewerCertificate = amazoncloudfront.ViewerCertificate{
			CertificateSource:            amazonaws.String("cloudfront"),
			CloudFrontDefaultCertificate: amazonaws.Bool(true),
			MinimumProtocolVersion:       amazonaws.String("SSLv3"),
		}
	}
	params := &amazoncloudfront.CreateDistributionInput{
		DistributionConfig: &amazoncloudfront.DistributionConfig{
			CallerReference: amazonaws.String(options.Domain),
			Comment:         amazonaws.String(options.Domain),
			DefaultCacheBehavior: &amazoncloudfront.DefaultCacheBehavior{
				ForwardedValues: &amazoncloudfront.ForwardedValues{
					Cookies: &amazoncloudfront.CookiePreference{
						Forward: amazonaws.String("none"),
					},
					QueryString: amazonaws.Bool(false),
					Headers: &amazoncloudfront.Headers{
						Quantity: amazonaws.Int64(0),
					},
				},
				MinTTL:         amazonaws.Int64(0),
				TargetOriginId: amazonaws.String("S3-" + options.Domain),
				TrustedSigners: &amazoncloudfront.TrustedSigners{
					Enabled:  amazonaws.Bool(false),
					Quantity: amazonaws.Int64(0),
				},
				ViewerProtocolPolicy: amazonaws.String("allow-all"),
				AllowedMethods: &amazoncloudfront.AllowedMethods{
					Items: []*string{
						amazonaws.String("HEAD"),
						amazonaws.String("GET"),
					},
					Quantity: amazonaws.Int64(2),
					CachedMethods: &amazoncloudfront.CachedMethods{
						Items: []*string{
							amazonaws.String("HEAD"),
							amazonaws.String("GET"),
						},
						Quantity: amazonaws.Int64(2),
					},
				},
				Compress:        amazonaws.Bool(false),
				DefaultTTL:      amazonaws.Int64(86400),
				MaxTTL:          amazonaws.Int64(31536000),
				SmoothStreaming: amazonaws.Bool(false),
			},
			Enabled: amazonaws.Bool(true),
			Origins: &amazoncloudfront.Origins{
				Quantity: amazonaws.Int64(1),
				Items: []*amazoncloudfront.Origin{
					{
						DomainName: amazonaws.String(options.Domain + ".s3-website-" + options.AWSRegion + ".amazonaws.com"),
						Id:         amazonaws.String("S3-" + options.Domain),
						CustomHeaders: &amazoncloudfront.CustomHeaders{
							Quantity: amazonaws.Int64(0),
						},
						CustomOriginConfig: &amazoncloudfront.CustomOriginConfig{
							HTTPPort:             amazonaws.Int64(80),
							HTTPSPort:            amazonaws.Int64(443),
							OriginProtocolPolicy: amazonaws.String("http-only"),
							OriginSslProtocols: &amazoncloudfront.OriginSslProtocols{
								Items: []*string{
									amazonaws.String("SSLv3"),
									amazonaws.String("TLSv1"),
								},
								Quantity: amazonaws.Int64(2),
							},
						},
						OriginPath: amazonaws.String(""),
					},
				},
			},
			Aliases: &amazoncloudfront.Aliases{
				Quantity: amazonaws.Int64(1),
				Items: []*string{
					amazonaws.String(options.Domain),
				},
			},
			CacheBehaviors: &amazoncloudfront.CacheBehaviors{
				Quantity: amazonaws.Int64(0),
			},
			CustomErrorResponses: &amazoncloudfront.CustomErrorResponses{
				Quantity: amazonaws.Int64(2),
				Items: []*amazoncloudfront.CustomErrorResponse{
					{
						ErrorCode:          amazonaws.Int64(403),
						ErrorCachingMinTTL: amazonaws.Int64(60),
						ResponseCode:       amazonaws.String("200"),
						ResponsePagePath:   amazonaws.String("/index.html"),
					},
					{
						ErrorCode:          amazonaws.Int64(404),
						ErrorCachingMinTTL: amazonaws.Int64(60),
						ResponseCode:       amazonaws.String("200"),
						ResponsePagePath:   amazonaws.String("/index.html"),
					},
				},
			},
			DefaultRootObject: amazonaws.String("index.html"),
			Logging: &amazoncloudfront.LoggingConfig{
				Bucket:         amazonaws.String(""),
				Enabled:        amazonaws.Bool(false),
				IncludeCookies: amazonaws.Bool(false),
				Prefix:         amazonaws.String(""),
			},
			PriceClass: amazonaws.String("PriceClass_All"),
			Restrictions: &amazoncloudfront.Restrictions{
				GeoRestriction: &amazoncloudfront.GeoRestriction{
					Quantity:        amazonaws.Int64(0),
					RestrictionType: amazonaws.String("none"),
				},
			},
			ViewerCertificate: &viewerCertificate,
			WebACLId:          amazonaws.String(""),
		},
	}
	resp, err := cloudfrontService.CreateDistribution(params)
	if err != nil {
		return "", err
	}

	fmt.Println("Creating a new CloudFront distribution with the bucket name.")
	return *resp.Distribution.DomainName, nil
}

/*
* Create new IAM user upon using the 'create' command, '--no-user' flag disables this
 */
func CreateUser(options Options) (key iam.AccessKey, err error) {
	name := options.Domain + "_deploy"

	_, err = iamSession.CreateUser(name, "/")
	if err != nil {
		iamErr, ok := err.(*iam.Error)
		if !ok || iamErr.Code != "EntityAlreadyExists" {
			return
		}
	}

	// user policy that only allows access to the specified bucket
	_, err = iamSession.PutUserPolicy(name, name, `{
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
						"arn:aws:s3:::`+options.Domain+`", "arn:aws:s3:::`+options.Domain+`/*"
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

/*
* Add Route53 route
 */
func UpdateRoute(options Options, distDomainName string) error {
	//get zone name
	zoneName, err := publicsuffix.EffectiveTLDPlusOne(options.Domain)
	if err != nil {
		return err
	}

	zoneName = zoneName + "."

	resp, err := r53Session.ListHostedZonesByName(zoneName, "", 100)
	if err != nil {
		return err
	}

	if resp.IsTruncated {
		panic("More than 100 zones in the account")
	}

	//find the first zone that matches the bucket zone name
	var zone *route53.HostedZone
	for _, z := range resp.HostedZones {
		if z.Name == zoneName {
			zone = &z
			break
		}
	}

	if zone == nil {
		fmt.Printf("A Route 53 hosted zone was not found for %s\n", zoneName)
		if zoneName != options.Domain {
			// the bucket could not be found in route53 and is a subdomain
			fmt.Println("If you would like to use Route 53 to manage your DNS, create a zone for this domain, and update your registrar's configuration to point to the DNS servers Amazon provides and rerun this command.  Note that you must copy any existing DNS configuration you have to Route 53 if you do not wish existing services hosted on this domain to stop working.")
			fmt.Printf("If you would like to continue to use your existing DNS, create a CNAME record pointing %s to %s and the site setup will be finished.\n", options.Domain, distDomainName)
		} else {
			//the bucket is not a subdomain
			// TODO(renandincer): Simplify this, it might be confusing to a user
			fmt.Println("Since you are hosting the root of your domain, using an alternative DNS host is unfortunately not possible.")
			fmt.Println("If you wish to host your site at the root of your domain, you must switch your sites DNS to Amazon's Route 53 and retry this command.")
		}

		return nil
	}

	fmt.Printf("Adding %s to %s Route 53 zone\n", options.Domain, zone.Name)
	parts := strings.Split(zone.Id, "/")
	idValue := parts[2]

	_, err = r53Session.ChangeResourceRecordSet(&route53.ChangeResourceRecordSetsRequest{
		Changes: []route53.Change{
			route53.Change{
				Action: "CREATE",
				Name:   options.Domain,
				Type:   "A",
				AliasTarget: route53.AliasTarget{
					HostedZoneId:         "Z2FDTNDATAQYW2", //cloudfront distribution
					DNSName:              distDomainName,
					EvaluateTargetHealth: false,
				},
			},
		},
	}, idValue)

	if err != nil {
		if strings.Contains(err.Error(), "it already exists") {
			fmt.Println("Existing route found, assuming it is correct")
			fmt.Printf("If you run into trouble, you may need to delete the %s route in Route53 and try again\n", options.Domain)
			return nil
		}
		return err
	}

	return nil
}

/*
* Find the best matching certificates to the domain name.
* return the best matching certificate if a certificate exists
* (see https://github.com/EagerIO/Stout/issues/20#issuecomment-232174716)
* or, if there is no certificate with the domain requested, return ""
 */
func findMatchingCertificate(options Options, acmService *acm.ACM) (string, error) {

	// list issued and pending certificates
	certificatesResponse, err := acmService.ListCertificates(&acm.ListCertificatesInput{
		CertificateStatuses: []*string{
			amazonaws.String("ISSUED"),
			amazonaws.String("PENDING_VALIDATION"),
		},
	})
	if err != nil {
		return "", err
	}
	//if there are no certificates, return nil
	if len(certificatesResponse.CertificateSummaryList) == 0 {
		return "", nil
	}

	// if --create-ssl is specified, only look for exact match and stout tag, otherwise return nil and
	if options.CreateSSL {
		//determine if any certificate is an exact match
		for _, certificate := range certificatesResponse.CertificateSummaryList {
			domainName := *certificate.DomainName

			certificateARN := *certificate.CertificateArn
			if domainName == options.Domain {
				tags, err := acmService.ListTagsForCertificate(&acm.ListTagsForCertificateInput{
					CertificateArn: amazonaws.String(certificateARN),
				})

				if err != nil {
					return "", err
				}

				for _, tag := range tags.Tags {
					if *tag.Key == "stout" && *tag.Value == "true" {
						//this cert was created by stout, take it!
						return certificateARN, nil
					}
				}
			}
		}
		return "", nil
	}

	//pick all suitable certificates
	for _, certificate := range certificatesResponse.CertificateSummaryList {
		//request certificate details
		domainName := *certificate.DomainName
		certificateARN := *certificate.CertificateArn

		//request certificate detail for all
		certificateDetail, err := acmService.DescribeCertificate(&acm.DescribeCertificateInput{
			CertificateArn: amazonaws.String(certificateARN),
		})

		if err != nil {
			return "", err
		}

		//domain name exactly matches the cert name
		if domainName == options.Domain {
			return certificateARN, nil
		}

		for _, certSAN := range certificateDetail.Certificate.SubjectAlternativeNames {
			if *certSAN == options.Domain {
				return certificateARN, nil
			}
		}

		//domain name falls under a wildcard domain
		wildcardDomainTLDPlusOne, err := publicsuffix.EffectiveTLDPlusOne(options.Domain)
		if err != nil {
			return "", err
		}
		wildCardOfGivenDomain := strings.Join([]string{"*", wildcardDomainTLDPlusOne}, ".")

		if wildCardOfGivenDomain == domainName {
			return certificateARN, nil
		}

	}
	return "", nil
}

/*
* Check if the certificate is issued(confirmed)
 */
func validateCertificate(acmService *acm.ACM, certificateARN string) error {
	certificateDetail, err := acmService.DescribeCertificate(&acm.DescribeCertificateInput{
		CertificateArn: amazonaws.String(certificateARN),
	})

	if err != nil {
		return err
	}

	if *certificateDetail.Certificate.Status != "ISSUED" {
		//the certificate was not issued yet prompt to check their email
		validationEmails := make([]string, 0)
		for _, validationOption := range certificateDetail.Certificate.DomainValidationOptions {
			for _, validationEmail := range validationOption.ValidationEmails {

				validationEmails = append(validationEmails, *validationEmail)
			}
		}
		vEmails := strings.Join(validationEmails, "\n\t- ")
		return fmt.Errorf("Certificate not issued yet. Please check one of the following emails or use --no-ssl:\n\t- %s", vEmails)
	} else {
		return nil
	}
}

/*
* request a new certificate from ACM
 */
func requestCertificate(options Options, acmService *acm.ACM) ([]string, error) {
	certificateReqResponse, err := acmService.RequestCertificate(&acm.RequestCertificateInput{
		DomainName: amazonaws.String(options.Domain),
		DomainValidationOptions: []*acm.DomainValidationOption{
			{
				DomainName:       amazonaws.String(options.Domain),
				ValidationDomain: amazonaws.String(options.Domain),
			},
		},
	})

	if err != nil {
		return nil, err
	}

	_, err = acmService.AddTagsToCertificate(&acm.AddTagsToCertificateInput{
		CertificateArn: amazonaws.String(*certificateReqResponse.CertificateArn),
		Tags: []*acm.Tag{
			{
				Key:   amazonaws.String("stout"),
				Value: amazonaws.String("true"),
			},
		},
	})

	if err != nil {
		return nil, err
	}

	var certificateDetail *acm.DescribeCertificateOutput
	// try getting a response from AWS multiple times as validation emails are not available immediately
	trials := 0
	for trials < 5 {
		time.Sleep(2 * time.Second)

		certificateDetail, err = acmService.DescribeCertificate(&acm.DescribeCertificateInput{
			CertificateArn: amazonaws.String(*certificateReqResponse.CertificateArn),
		})

		if err != nil {
			return nil, err
		}
		if len(certificateDetail.Certificate.DomainValidationOptions[0].ValidationEmails) != 0 {
			break
		} else {
			trials++
		}
	}
	validationEmails := make([]string, 0)

	for _, validationOption := range certificateDetail.Certificate.DomainValidationOptions {
		for _, validationEmail := range validationOption.ValidationEmails {
			validationEmails = append(validationEmails, *validationEmail)
		}
	}
	return validationEmails, nil
}

/*
* Set up ssl/tls certificates
 */
func setUpSSL(options Options, awsSession *session.Session) (string, error) {
	if options.CreateSSL && options.NoSSL {
		return "", errors.New("You have specified conflicting options: please choose either --no-ssl or --create-ssl.")
	}

	// if the person wants ssl certificates
	if !options.NoSSL {
		acmService := acm.New(awsSession)
		certificateARN, err := findMatchingCertificate(options, acmService)

		if err != nil {
			return "", errors.New("Could not list ACM certificates while trying to find one to use")
		}

		// if there is a certificate found
		if certificateARN != "" {
			//is there a certificate
			err := validateCertificate(acmService, certificateARN)

			if err != nil {
				return "", err
			}
			fmt.Printf("Using certificate with ARN: %q\n", certificateARN)
			return certificateARN, nil
		} else {
			// no certificate was found, create or ask the user
			if options.CreateSSL {
				fmt.Println("No certificate found to use, creating a new one.")
				validationEmails, err := requestCertificate(options, acmService)
				if err != nil {
					return "", err
				}
				errorText := fmt.Sprintf("Please check one of the email addresses below to confirm your new SSL/TLS certificate and run this command again. \n\t- %s", strings.Join(validationEmails, "\n\t- "))
				return "", errors.New(errorText)
			} else {
				// no certificate wes found and nothing was specified.
				// have a conversation with the user asking what they want to do
				// or if it it headless, don't set up a cert
				if terminal.IsTerminal(int(os.Stdout.Fd())) {
					// talk to the user
					errorText := fmt.Sprintf("Please specify if you'd like a ssl certificate or not: %q or %q", "--create-ssl", " --no-ssl")
					return "", errors.New(errorText)
				} else {
					// set up without ssl
					return "", nil
				}
			}
		}
	}
	// --no-ssl
	return "", nil
}

/*
* Create a new user, CloudFront distrbution, s3 bucket, route53 route
 */
func Create(options Options) {
	_, err := exec.LookPath("aws")
	if err != nil {
		fmt.Println("The aws CLI executable was not found in the PATH")
		fmt.Println("Install it from http://aws.amazon.com/cli/ and try again")
	}

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

	//official sdk connection
	if awsSession == nil {
		awsSession = session.New(&amazonaws.Config{
			Region:      amazonaws.String(options.AWSRegion),
			Credentials: credentials.NewStaticCredentials(options.AWSKey, options.AWSSecret, ""),
		})
	}
	fmt.Println("Checking for available SSL/TLS certificates")
	certificateARN, err := setUpSSL(options, awsSession)
	if err != nil {
		fmt.Println("Error while processing SSL/TLS certificates")
		fmt.Println(err)
		return
	}
	if certificateARN == "" {
		fmt.Println("Will set up CloudFront distrbution without SSL/TLS")
	}

	fmt.Println("Creating Bucket")
	err = CreateBucket(options)

	if err != nil {
		fmt.Println("Error creating S3 bucket")
		fmt.Println(err)
		return
	}
	fmt.Println("Loading/Creating CloudFront Distribution")
	distDomainName, err := GetDistribution(awsSession, options, certificateARN)

	if err != nil {
		fmt.Println("Error loading/creating CloudFront distribution")
		fmt.Println(err)
		return
	}

	fmt.Println("Adding Route")
	err = UpdateRoute(options, distDomainName)

	if err != nil {
		fmt.Println("Error adding route to Route53 DNS config")
		fmt.Println(err)
		return
	}

	if !options.NoUser {
		key, err := CreateUser(options)

		if err != nil {
			fmt.Println("Error creating user")
			fmt.Println(err)
			return
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

	stout deploy --domain ` + options.Domain + ` --key ` + key.Id + ` --secret '` + key.Secret + `'
`)
		}

	}

	fmt.Println("You can begin deploying now, but it can take up to twenty minutes for your site to begin to work")
	fmt.Println("Depending on the configuration of your site, you might need to set the 'root', 'dest' or 'files' options to get your deploys working as you wish.  See the README for details.")
	fmt.Println("It's also a good idea to look into the 'env' option, as in real-world situations it usually makes sense to have a development and/or staging site for each of your production sites.")
}
