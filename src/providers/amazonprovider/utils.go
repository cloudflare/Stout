package amazonprovider

import (
	"errors"
	"os/exec"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/zackbloom/goamz/aws"
	"github.com/zackbloom/goamz/cloudfront"
	"github.com/zackbloom/goamz/iam"
	"github.com/zackbloom/goamz/route53"
	"github.com/zackbloom/goamz/s3"
)

func checkForAWS() error {
	_, err := exec.LookPath("aws")
	if err != nil {
		return errors.New("The aws CLI executable was not found in the PATH\n" +
			"Install it from http://aws.amazon.com/cli/ and try again")
	}

	return nil
}

const (
	LIMITED = 60
	FOREVER = 31556926
)

var s3Session *s3.S3
var iamSession *iam.IAM
var r53Session *route53.Route53
var cfSession *cloudfront.CloudFront

var awsSession *session.Session

/*
* Check is the specified region is a valid region
 */
func getRegion(region string) aws.Region {
	regionS, ok := aws.Regions[region]
	if !ok {
		panic("Region not found")
	}
	return regionS
}

/*
*	Open a new S3 connection
 */
func openS3(key, secret, region string) *s3.S3 {
	regionS := getRegion(region)

	auth := aws.Auth{
		AccessKey: key,
		SecretKey: secret,
	}
	return s3.New(auth, regionS)
}

/*
*	Open a new IAM connection
 */
func openIAM(key, secret, region string) *iam.IAM {
	regionS := getRegion(region)

	auth := aws.Auth{
		AccessKey: key,
		SecretKey: secret,
	}
	return iam.New(auth, regionS)
}

/*
*	Open a new CF connection
 */
func openCloudFront(key, secret string) *cloudfront.CloudFront {
	auth := aws.Auth{
		AccessKey: key,
		SecretKey: secret,
	}
	return cloudfront.NewCloudFront(auth)
}

/*
*	Open a new Route53 connection
 */
func openRoute53(key, secret string) *route53.Route53 {
	auth := aws.Auth{
		AccessKey: key,
		SecretKey: secret,
	}

	r53, _ := route53.NewRoute53(auth)
	return r53
}
