# amazon

The amazon provider supports FS through S3, CDN through Cloudfront, and DNS through Route53.

The amazon provider package checks first for key/secret combos in `~/.aws/config` and `~/.aws/credentials`, which are overridden by the config file and CLI flags, if they exist.

## Options

The `create-custom-ssl` boolean flag creates a custom ssl certificate, provisioning it through AWS. This means that instead of using the `*.cloudfront.net` default certificate, AWS provisions you a custom SSL certificate with the common name as your domain, e.g. `*.example.com`.

The `new-user` boolean flag creates a new AWS user with only the permissions required to deploy/rollback the file storage, and it's recommended to use this user for future deploys/rollbacks for increased security.

## Config

The providers section of an example config file using amazon could look like the following:

```yaml
[...]
        providers:
                amazon:
                        key: "testkey"
                        secret: "testsecret"
                        region: "us-east-1"
                        new-user: true
                        create-custom-ssl: true
```


# Advanced

### Custom self-provisioned SSL

Cloudfront also has the ability to serve your site using SSL certificates that you've provisioned from somewhere other than AWS. The general procedure for setting it up is:

1. Get an SSL certificate for your domain
2. Upload it to Amazon
3. Select that certificate in the configuration for the CloudFront distribution Stout creates for you

You will absolutely need more detailed instructions, which you can find [here](https://bryce.fisher-fleig.org/blog/setting-up-ssl-on-aws-cloudfront-and-s3/).

Selecting a certificate for you is one of the few things the `create` command does not do, as it's not always possible to decide which certificate is appropriate.

## AWS user permissions

The AWS user which is used for Stout should have the `GetObject`, `PutObject`, `DeleteObject`, and `ListBucket` permissions. The `create` command used with the `new-user` flag will set this up for you if you use it.

This is an example policy config which works:

```json
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
        "arn:aws:s3:::BUCKET", "arn:aws:s3:::BUCKET/*"
      ]
    }
  ]
}
```

Be sure to replace `BUCKET` with your bucket's actual name.
