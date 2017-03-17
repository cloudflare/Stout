package amazonprovider

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/net/publicsuffix"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acm"
	cloudfront "github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/zackbloom/goamz/iam"
	"github.com/zackbloom/goamz/route53"
	"github.com/zackbloom/goamz/s3"
)

func CreateBucket(domain string) error {
	bucket := s3Session.Bucket(domain)

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
					"Resource": "arn:aws:s3:::` + domain + `/*"
				}
			]
		}`,
	))
	if err != nil {
		return err
	}

	return nil
}

// Get cloudfront distribution
// create one if none already exists for the specified domain name in options
// Check if SSL is set up for the distribution and set that up if a ACM Cert ARN is passed in
// Returns distribution domain name
func GetDistribution(amazonSession *session.Session, certificateARN string, createSSL bool, domain string, awsRegion string) (string, error) {
	cloudfrontService := cloudfront.New(amazonSession)
	dists, err := cloudfrontService.ListDistributions(&cloudfront.ListDistributionsInput{
		MaxItems: aws.Int64(500),
	})
	if err != nil {
		return "", err
	}

	if *dists.DistributionList.IsTruncated {
		return "", errors.New("You have more than 500 distributions, please consider opening a GitHub issue if you require support for this.")
	}

	var currentDist *cloudfront.DistributionSummary
	//check for already existing distributions
	for _, distSummary := range dists.DistributionList.Items {
		for _, alias := range distSummary.Aliases.Items {
			if *alias == domain {
				//matching distribution found
				fmt.Println("CloudFront distribution found with the provided bucket name, assuming config matches.")
				fmt.Println("If you run into issues, delete the distribution and rerun this command.")
				currentDist = distSummary
			}
		}
	}

	if currentDist != nil && !createSSL {
		// if there is a distribution already and there is no upgrade request, return it
		return *currentDist.DomainName, nil
	} else if currentDist != nil && createSSL {
		// if there is a distribution and createssl is an option, check if the distribution has a certificate
		distDetail, err := cloudfrontService.GetDistribution(&cloudfront.GetDistributionInput{
			Id: aws.String(*currentDist.Id),
		})
		if err != nil {
			return "", err
		}
		// if there is no certificate installed, update it
		if *distDetail.Distribution.DistributionConfig.ViewerCertificate.CloudFrontDefaultCertificate {
			fmt.Println("Updating current CloudFront distribution with your new certificate")
			distDetail.Distribution.DistributionConfig.ViewerCertificate = &cloudfront.ViewerCertificate{
				ACMCertificateArn:      aws.String(certificateARN),
				Certificate:            aws.String(certificateARN),
				CertificateSource:      aws.String("acm"),
				MinimumProtocolVersion: aws.String("TLSv1"),
				SSLSupportMethod:       aws.String("sni-only"),
			}
			_, err := cloudfrontService.UpdateDistribution(&cloudfront.UpdateDistributionInput{
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
	var viewerCertificate cloudfront.ViewerCertificate
	if certificateARN != "" {
		viewerCertificate = cloudfront.ViewerCertificate{
			ACMCertificateArn:      aws.String(certificateARN),
			Certificate:            aws.String(certificateARN),
			CertificateSource:      aws.String("acm"),
			MinimumProtocolVersion: aws.String("TLSv1"),
			SSLSupportMethod:       aws.String("sni-only"),
		}
	} else {
		viewerCertificate = cloudfront.ViewerCertificate{
			CertificateSource:            aws.String("cloudfront"),
			CloudFrontDefaultCertificate: aws.Bool(true),
			MinimumProtocolVersion:       aws.String("SSLv3"),
		}
	}
	params := &cloudfront.CreateDistributionInput{
		DistributionConfig: &cloudfront.DistributionConfig{
			CallerReference: aws.String(domain),
			Comment:         aws.String(domain),
			DefaultCacheBehavior: &cloudfront.DefaultCacheBehavior{
				ForwardedValues: &cloudfront.ForwardedValues{
					Cookies: &cloudfront.CookiePreference{
						Forward: aws.String("none"),
					},
					QueryString: aws.Bool(false),
					Headers: &cloudfront.Headers{
						Quantity: aws.Int64(0),
					},
				},
				MinTTL:         aws.Int64(0),
				TargetOriginId: aws.String("S3-" + domain),
				TrustedSigners: &cloudfront.TrustedSigners{
					Enabled:  aws.Bool(false),
					Quantity: aws.Int64(0),
				},
				ViewerProtocolPolicy: aws.String("allow-all"),
				AllowedMethods: &cloudfront.AllowedMethods{
					Items: []*string{
						aws.String("HEAD"),
						aws.String("GET"),
					},
					Quantity: aws.Int64(2),
					CachedMethods: &cloudfront.CachedMethods{
						Items: []*string{
							aws.String("HEAD"),
							aws.String("GET"),
						},
						Quantity: aws.Int64(2),
					},
				},
				Compress:        aws.Bool(false),
				DefaultTTL:      aws.Int64(86400),
				MaxTTL:          aws.Int64(31536000),
				SmoothStreaming: aws.Bool(false),
			},
			Enabled: aws.Bool(true),
			Origins: &cloudfront.Origins{
				Quantity: aws.Int64(1),
				Items: []*cloudfront.Origin{
					{
						DomainName: aws.String(domain + ".s3-website-" + awsRegion + ".aws.com"),
						Id:         aws.String("S3-" + domain),
						CustomHeaders: &cloudfront.CustomHeaders{
							Quantity: aws.Int64(0),
						},
						CustomOriginConfig: &cloudfront.CustomOriginConfig{
							HTTPPort:             aws.Int64(80),
							HTTPSPort:            aws.Int64(443),
							OriginProtocolPolicy: aws.String("http-only"),
							OriginSslProtocols: &cloudfront.OriginSslProtocols{
								Items: []*string{
									aws.String("SSLv3"),
									aws.String("TLSv1"),
								},
								Quantity: aws.Int64(2),
							},
						},
						OriginPath: aws.String(""),
					},
				},
			},
			Aliases: &cloudfront.Aliases{
				Quantity: aws.Int64(1),
				Items: []*string{
					aws.String(domain),
				},
			},
			CacheBehaviors: &cloudfront.CacheBehaviors{
				Quantity: aws.Int64(0),
			},
			CustomErrorResponses: &cloudfront.CustomErrorResponses{
				Quantity: aws.Int64(2),
				Items: []*cloudfront.CustomErrorResponse{
					{
						ErrorCode:          aws.Int64(403),
						ErrorCachingMinTTL: aws.Int64(60),
						ResponseCode:       aws.String("200"),
						ResponsePagePath:   aws.String("/index.html"),
					},
					{
						ErrorCode:          aws.Int64(404),
						ErrorCachingMinTTL: aws.Int64(60),
						ResponseCode:       aws.String("200"),
						ResponsePagePath:   aws.String("/index.html"),
					},
				},
			},
			DefaultRootObject: aws.String("index.html"),
			Logging: &cloudfront.LoggingConfig{
				Bucket:         aws.String(""),
				Enabled:        aws.Bool(false),
				IncludeCookies: aws.Bool(false),
				Prefix:         aws.String(""),
			},
			PriceClass: aws.String("PriceClass_All"),
			Restrictions: &cloudfront.Restrictions{
				GeoRestriction: &cloudfront.GeoRestriction{
					Quantity:        aws.Int64(0),
					RestrictionType: aws.String("none"),
				},
			},
			ViewerCertificate: &viewerCertificate,
			WebACLId:          aws.String(""),
		},
	}
	resp, err := cloudfrontService.CreateDistribution(params)
	if err != nil {
		return "", err
	}

	fmt.Println("Creating a new CloudFront distribution with the bucket name.")
	return *resp.Distribution.DomainName, nil
}

// Create new IAM user upon using the 'create' command, '--no-user' flag disables this
func CreateUser(domain string) (key iam.AccessKey, err error) {
	name := domain + "_deploy"

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
						"arn:aws:s3:::`+domain+`", "arn:aws:s3:::`+domain+`/*"
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
func UpdateRoute(domain string, distDomainName string) error {
	//get zone name
	zoneName, err := publicsuffix.EffectiveTLDPlusOne(domain)
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
		if zoneName != domain {
			// the bucket could not be found in route53 and is a subdomain
			fmt.Println("If you would like to use Route 53 to manage your DNS, create a zone for this domain, and update your registrar's configuration to point to the DNS servers Amazon provides and rerun this command.  Note that you must copy any existing DNS configuration you have to Route 53 if you do not wish existing services hosted on this domain to stop working.")
			fmt.Printf("If you would like to continue to use your existing DNS, create a CNAME record pointing %s to %s and the site setup will be finished.\n", domain, distDomainName)
		} else {
			//the bucket is not a subdomain
			// TODO(renandincer): Simplify this, it might be confusing to a user
			fmt.Println("Since you are hosting the root of your domain, using an alternative DNS host is unfortunately not possible.")
			fmt.Println("If you wish to host your site at the root of your domain, you must switch your sites DNS to Amazon's Route 53 and retry this command.")
		}

		return nil
	}

	fmt.Printf("Adding %s to %s Route 53 zone\n", domain, zone.Name)
	parts := strings.Split(zone.Id, "/")
	idValue := parts[2]

	_, err = r53Session.ChangeResourceRecordSet(&route53.ChangeResourceRecordSetsRequest{
		Changes: []route53.Change{
			route53.Change{
				Action: "CREATE",
				Name:   domain,
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
			fmt.Printf("If you run into trouble, you may need to delete the %s route in Route53 and try again\n", domain)
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
func findMatchingCertificate(acmService *acm.ACM, domain string, createSSL bool) (string, error) {

	// list issued and pending certificates
	certificatesResponse, err := acmService.ListCertificates(&acm.ListCertificatesInput{
		CertificateStatuses: []*string{
			aws.String("ISSUED"),
			aws.String("PENDING_VALIDATION"),
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
	if createSSL {
		//determine if any certificate is an exact match
		for _, certificate := range certificatesResponse.CertificateSummaryList {
			domainName := *certificate.DomainName

			certificateARN := *certificate.CertificateArn
			if domainName == domain {
				tags, err := acmService.ListTagsForCertificate(&acm.ListTagsForCertificateInput{
					CertificateArn: aws.String(certificateARN),
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
			CertificateArn: aws.String(certificateARN),
		})

		if err != nil {
			return "", err
		}

		//domain name exactly matches the cert name
		if domainName == domain {
			return certificateARN, nil
		}

		for _, certSAN := range certificateDetail.Certificate.SubjectAlternativeNames {
			if *certSAN == domain {
				return certificateARN, nil
			}
		}

		//domain name falls under a wildcard domain
		wildcardDomainTLDPlusOne, err := publicsuffix.EffectiveTLDPlusOne(domain)
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
		CertificateArn: aws.String(certificateARN),
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
func requestCertificate(acmService *acm.ACM, domain string) ([]string, error) {
	certificateReqResponse, err := acmService.RequestCertificate(&acm.RequestCertificateInput{
		DomainName: aws.String(domain),
		DomainValidationOptions: []*acm.DomainValidationOption{
			{
				DomainName:       aws.String(domain),
				ValidationDomain: aws.String(domain),
			},
		},
	})

	if err != nil {
		return nil, err
	}

	_, err = acmService.AddTagsToCertificate(&acm.AddTagsToCertificateInput{
		CertificateArn: aws.String(*certificateReqResponse.CertificateArn),
		Tags: []*acm.Tag{
			{
				Key:   aws.String("stout"),
				Value: aws.String("true"),
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
			CertificateArn: aws.String(*certificateReqResponse.CertificateArn),
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
func setUpSSL(awsSession *session.Session, domain string, createSSL bool, noSSL bool) (string, error) {
	if createSSL && noSSL {
		return "", errors.New("You have specified conflicting options: please choose either --no-ssl or --create-ssl.")
	}

	// if the person wants ssl certificates
	if !noSSL {
		acmService := acm.New(awsSession)
		certificateARN, err := findMatchingCertificate(acmService, domain, createSSL)

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
			if createSSL {
				fmt.Println("No certificate found to use, creating a new one.")
				validationEmails, err := requestCertificate(acmService, domain)
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
