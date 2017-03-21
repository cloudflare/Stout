package fs

import (
	"github.com/zackbloom/goamz/iam"
	"github.com/zackbloom/goamz/s3"
)

// Create new IAM user upon using the 'create' command, '--no-user' flag disables this
func CreateS3User(domain string) (key iam.AccessKey, err error) {
	name := domain + "_deploy"

	_, err = IamSession.CreateUser(name, "/")
	if err != nil {
		iamErr, ok := err.(*iam.Error)
		if !ok || iamErr.Code != "EntityAlreadyExists" {
			return
		}
	}

	// user policy that only allows access to the specified bucket
	_, err = IamSession.PutUserPolicy(name, name, `{
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

	keyResp, err := IamSession.CreateAccessKey(name)
	if err != nil {
		return
	}

	return keyResp.AccessKey, nil
}

func CreateS3Bucket(domain string) error {
	bucket := S3Session.Bucket(domain)

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
