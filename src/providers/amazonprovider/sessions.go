package amazonprovider

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudfront"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/s3"
)

const (
	LIMITED = 60
	FOREVER = 31556926
)

var awsSession *session.Session

var s3Session *s3.S3
var iamSession *iam.IAM
var r53Session *route53.Route53
var cfSession *cloudfront.CloudFront

func setupAWS(key, secret, region string) error {
	creds := credentials.NewStaticCredentials(key, secret, "")
	config := (&aws.Config{}).WithCredentials(creds)

	awsSession = session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config:            *config,
	}))

	s3Session = s3.New(awsSession)
	iamSession = iam.New(awsSession)
	r53Session = route53.New(awsSession)
	cfSession = cloudfront.New(awsSession)

	return nil
}
