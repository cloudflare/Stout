### Setting Up Your Site

If you don't already have an S3-hosted site, start here.

We're going to create a basic site config which uses CloudFront's CDN to deliver high performance at a minimal cost.  Once you run the setup, you'll end up with a configuration which looks like this:

```
The Deploy Tool -> S3 <- CloudFront's Global CDN <- DNS <- Your Users
```

The simplest way to get started is to run the `create_site.sh` script in the utils folder.  After installing the [aws command line tools](http://aws.amazon.com/cli/), run:

```bash
./utils/create_site.sh subdomain.my-site.com
```

Feel free to leave out the subdomain if you'd like to host it at the root of your domain.

This will:

- Create an S3 bucket for this site with the correct security policy and website hosting
- Create a CloudFront distribution pointed at that bucket/domain
- Create a user with the appropriate permissions to upload to that bucket
- Create an access key for that user

Once that's done, copy the access key, secret key (from the JSON blob the access key request spits out) and domain (the bucket's name is just the domain you provided) it printed to your `deploy.yaml`, or save them to use with the `stout deploy` as arguments.

The final step is to point your DNS records to the new CloudFront distribution.  If you use Route 53 you want to create an alias to the distribution (it will be named the same as the new domain).  If you use another DNS service, you'll want to create a CNAME to the CloudFront distribution's hostname.

Please note that it will take up to twenty minutes for the CloudFront distribution to initialize.  Additionally it may take some time for your DNS records to update.

If you'd like development or staging environments, just run the command again with the URL you'd like them to have, and add the new credentials as before.  See the "YAML Config" section of the README for an example of how to configure multiple environments.

Be very careful to never commit a file to a public repo that contains your AWS credentials.  If you are deploying a public repo, either keep the credentials on your local machine you deploy from, or in the build service (like CircleCI) you're using.

#### Step-by-step Instructions

1. Install Amazon's AWS Command-Line Tools (and create an AWS account if you don't have one)
1. Run the `create_site.sh` tool with the URL of the site you'd like to deploy
1. Take note of the AWS key and secret in the final JSON blob outputted by the script
1. Download the executable from this project
1. Run `stout deploy --domain subdomain.your-site.com --key YOUR_NEW_AWS_KEY --secret YOUR_NEW_AWS_SECRET` to deploy
1. Add the `--root` argument if your built files are in a subdirectory.
1. Visit the cloudfront url of your new distribution to see how your site currently looks, include any new files you may have missed and deploy again
1. Optionally, Move any configuration options you don't mind being committed to your repository to a deploy.yaml file
1. Optionally, Run `create_site.sh` again to create staging or development sites, and add their configuration to your deploy.yaml as well
1. Optionally, Deploy more projects to this same site by running deploy with the `--dest` argument
1. Optionally, Add the deploy step to your build tool
