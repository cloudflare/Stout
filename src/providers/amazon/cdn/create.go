package cdn

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudfront"
)

// Get cloudfront distribution
// create one if none already exists for the specified domain name in options
// Check if SSL is set up for the distribution and set that up if a ACM Cert ARN is passed in
// Returns distribution domain name
func GetCFDistribution(cfSession *cloudfront.CloudFront, certificateARN string, createSSL bool, domain string, awsRegion string) (string, error) {
	fmt.Println("Getting a CloudFront distribution with the domain name.")
	var currentDist *cloudfront.DistributionSummary

	var marker *string
	for {
		dists, err := cfSession.ListDistributions(&cloudfront.ListDistributionsInput{
			MaxItems: aws.Int64(500),
			Marker:   marker,
		})
		if err != nil {
			return "", err
		}

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
		if currentDist != nil {
			break
		}

		if *dists.DistributionList.IsTruncated {
			marker = dists.DistributionList.NextMarker
		} else {
			break
		}
	}

	if currentDist != nil {
		// if there is a distribution already and there is no upgrade request, return it
		if !createSSL {
			return *currentDist.DomainName, nil
		}

		// if the default certificate is installed (as opposed to a custom one), update it
		certIsDefault := currentDist.ViewerCertificate.CloudFrontDefaultCertificate
		if certIsDefault != nil && *certIsDefault {
			// get the actual distribution
			distDetail, err := cfSession.GetDistribution(&cloudfront.GetDistributionInput{
				Id: currentDist.Id,
			})
			if err != nil {
				return "", err
			}

			fmt.Println("Updating current CloudFront distribution with your new certificate")
			distDetail.Distribution.DistributionConfig.ViewerCertificate = &cloudfront.ViewerCertificate{
				ACMCertificateArn:      aws.String(certificateARN),
				Certificate:            aws.String(certificateARN),
				CertificateSource:      aws.String("acm"),
				MinimumProtocolVersion: aws.String("TLSv1"),
				SSLSupportMethod:       aws.String("sni-only"),
			}
			_, err = cfSession.UpdateDistribution(&cloudfront.UpdateDistributionInput{
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

	fmt.Println("Existing CloudFront distribution not found. Creating one.")
	resp, err := createCFDistribution(cfSession, certificateARN, domain, awsRegion)
	if err != nil {
		return "", err
	}

	return *resp.Distribution.DomainName, nil
}

func createCFDistribution(cfSession *cloudfront.CloudFront, certificateARN string, domain string, awsRegion string) (*cloudfront.CreateDistributionOutput, error) {
	var viewerCertificate cloudfront.ViewerCertificate
	if certificateARN == "" {
		viewerCertificate = cloudfront.ViewerCertificate{
			CertificateSource:            aws.String("cloudfront"),
			CloudFrontDefaultCertificate: aws.Bool(true),
			MinimumProtocolVersion:       aws.String("SSLv3"),
		}
	} else {
		viewerCertificate = cloudfront.ViewerCertificate{
			ACMCertificateArn:      aws.String(certificateARN),
			Certificate:            aws.String(certificateARN),
			CertificateSource:      aws.String("acm"),
			MinimumProtocolVersion: aws.String("TLSv1"),
			SSLSupportMethod:       aws.String("sni-only"),
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
						ResponsePagePath:   aws.String("/404.html"),
					},
					{
						ErrorCode:          aws.Int64(404),
						ErrorCachingMinTTL: aws.Int64(60),
						ResponseCode:       aws.String("200"),
						ResponsePagePath:   aws.String("/404.html"),
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

	return cfSession.CreateDistribution(params)
}
