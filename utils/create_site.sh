#set -e

export HOST=$1
export DEPLOY_USER=${HOST}_deploy

aws s3 mb s3://$HOST --region us-east-1

aws s3 website s3://$HOST --index-document index.html --error-document error.html

aws s3api put-bucket-policy --bucket $HOST --policy "{
  \"Version\": \"2008-10-17\",
  \"Statement\": [
    {
      \"Sid\": \"PublicReadForGetBucketObjects\",
      \"Effect\": \"Allow\",
      \"Principal\": {
        \"AWS\": \"*\"
      },
      \"Action\": \"s3:GetObject\",
      \"Resource\": \"arn:aws:s3:::$HOST/*\"
    }
  ]
}"

export CALLER=`date +"%T"`

aws cloudfront create-distribution --distribution-config "
{
    \"CallerReference\": \"$CALLER\",
    \"Comment\": null,
    \"CacheBehaviors\": {
        \"Quantity\": 0
    },
    \"Logging\": {
        \"Bucket\": null,
        \"Prefix\": null,
        \"Enabled\": false,
        \"IncludeCookies\": false
    },
    \"Origins\": {
        \"Items\": [
            {
                \"S3OriginConfig\": {
                    \"OriginAccessIdentity\": null
                },
                \"Id\": \"S3-$HOST\",
                \"DomainName\": \"$HOST.s3.amazonaws.com\"
            }
        ],
        \"Quantity\": 1
    },
    \"DefaultRootObject\": \"index.html\",
    \"PriceClass\": \"PriceClass_All\",
    \"Enabled\": true,
    \"DefaultCacheBehavior\": {
        \"TrustedSigners\": {
            \"Enabled\": false,
            \"Quantity\": 0
        },
        \"TargetOriginId\": \"S3-$HOST\",
        \"ViewerProtocolPolicy\": \"allow-all\",
        \"ForwardedValues\": {
            \"Cookies\": {
                \"Forward\": \"none\"
            },
            \"QueryString\": false
        },
        \"AllowedMethods\": {
            \"Items\": [
                \"GET\",
                \"HEAD\"
            ],
            \"Quantity\": 2
        },
        \"MinTTL\": 0
    },
    \"ViewerCertificate\": {
        \"CloudFrontDefaultCertificate\": true
    },
    \"CustomErrorResponses\": {
        \"Quantity\": 0
    },
    \"Restrictions\": {
        \"GeoRestriction\": {
            \"RestrictionType\": \"none\",
            \"Quantity\": 0
        }
    },
    \"Aliases\": {
        \"Items\": [
            \"$HOST\"
        ],
        \"Quantity\": 1
    }
}"

aws iam create-user --user-name $DEPLOY_USER
aws iam put-user-policy --user-name $DEPLOY_USER --policy-name $DEPLOY_USER --policy-document "{
  \"Version\": \"2012-10-17\",
  \"Statement\": [
    {
      \"Effect\": \"Allow\",
      \"Action\": [
        \"s3:DeleteObject\",
        \"s3:ListBucket\",
        \"s3:PutObject\",
        \"s3:PutObjectAcl\",
        \"s3:GetObject\"
      ],
      \"Resource\": [
        \"arn:aws:s3:::$HOST\", \"arn:aws:s3:::$HOST/*\"
      ]
    }
  ]
}"

aws iam create-access-key --user-name $DEPLOY_USER | cat

echo "Select a SSL Cert in CloudFront if applicable"

echo "Site setup. You must now manually add the cloudfront distribution to your DNS configuration."
