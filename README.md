# Stout

#### [View the site](http://stout.is/)

Stout is a deploy tool for static websites.  It takes a website and uploads it to S3 in a more reliable way than other tools.  It can be an alternative
to paid services like Divshot, to dynamic web servers like Rails, or to manually uploading your site to S3 or an FTP server.

## Why You Need Stout

Traditionally uploading your files to S3 introduces a serious caching issue we ran into in practice at [Eager](https://eager.io).
The cache for the various files your site depends on can expire at different times, meaning your users get an inconsistent (broken) set of files for a
time after every single deploy.  Further, traditional static site deployments don't offer any method of rolling back a previous deploy.

We built Stout to fix these issues.

### Features

- Versions script and style files to ensure your pages don't use an inconsistent set of files during or after a deploy
- Supports rollback to any previous version
- Does not depend on any specific build tool or workflow (it is a standalone executable written in Go)
- Does not require a datastore of any kind to maintain state or history
- Can be used by multiple developers simultaneously without locking or a danger of inconsistent state
- Properly handles caching headers
- Supports deploying multiple projects to various subdirectories of the same site without conflicts
- Compresses files for faster delivery

### Limitations

- Stout doesn't currently support rolling back files that aren't HTML, JS or CSS (images, videos, etc.).  See the Versioning section for more information.
- All-or-nothing consistency is only guarenteed on a per-html-file basis, not for the entire deploy.  See the Consistency section for more information.

## Getting Started

[Download](https://github.com/EagerIO/Stout/releases) the `stout` executable for your system from our latest release into a directory on your `$PATH`, like `/usr/local/bin`.

You can use the `create` command to create a new site.  It automatically creates an S3 bucket, a CloudFront distribution, and a user account for deployment.  It therefore requires
credentials for an Amazon AWS account which has permissions to those services along with Route 53.

```sh
stout create --bucket my.website.com --key MY_AWS_KEY --secret MY_AWS_SECRET
```

You can then deploy your project:

```sh
stout deploy --bucket my.website.com --key MY_AWS_KEY --secret MY_AWS_SECRET
```

If your built files are in another directory, add the `--root` option:

```sh
stout deploy --bucket my.website.com --key MY_AWS_KEY --secret MY_AWS_SECRET --root ./build
```

If you don't want to deploy all the files in your folder, use the files argument.

```sh
stout deploy --bucket my.website.com --key MY_AWS_KEY --secret MY_AWS_SECRET --root ./build --files "*.html,images/*"
```

Javascript and CSS included in your HTML files will always be included automatically.

The deploy command will give you a deploy id you can use in the future to rollback if you have to:

```sh
stout rollback --bucket my.website.com --key MY_AWS_KEY --secret MY_AWS_SECRET a3b8ff290c33
```

Eventually you'll probably want to move your config to a deploy.yaml file, rather than specifying it in the command every time.

Using the info below you can learn about what the deploy/rollback tools actually do, deploying to subfolders, deploying from your build tool, and rolling back.

## Backstory

We wrote Stout because we couldn't find an open-source way to reliably and efficiently deploy our static sites (including our app and blog).  We used a traditional upload-to-s3 tool, but caching meant a user could get the new html and styles, but the old scripts, for example, causing sporatic and random errors.  It also didn't support reliably rolling back when necessary.  We built Stout to be the reliable, production-ready choice for static deploys.

## Function

Stout is an executable file built from Go code.  The `deploy` command deploys one or more html files and their dependencies to a specified location in S3.  The `rollback` command takes a deploy id and rolls the project back to that version.

### Deploy

The deploy process works by parsing the script and style tags out of one or more html files.  It then hashes those files, uploads them prefixed with their hashes, and updates the location of the original script and link tags with the hashed locations.

It generates a deploy id by hashing all of the files in the deploy, and uploads the html files to a location prefixed by the deploy id.

When the uploads are successful, the prefixed html files are atomically copied to their unprefixed paths, completing the deploy.

### Rollback

A rollback simply copies the html files prefixed with the specified deploy id to the unprefixed paths.

### Deploy Configuration

You can configure the deploy tool with any combination of command line flags or arguments provided in a configuration yaml file.

The options are:

##### `bucket`
  The S3 bucket to deploy to.  In most configurations this bucket should be the origin for the CDN which actually serves your site.  It usually makes sense to make this the url you are going to host your site from (i.e. `"example.com"`)
  	
##### `config` ("./deploy.yaml")
  The location of a yaml file to read any otherwise unspecified configuration from.
   
##### `dest` ("./")
  The destination directory to write files to in the S3 bucket.   For example if you wanted your this project to end up hosted at `yoursite.com/blog`, you would specify `--dest blog`.
  	
##### `root` ("./")
 The local directory where the files to be uploaded lives.  It's common to make this your "./build" directory or the like.
 
##### `files` ("*")
  Comma-seperated glob patterns of the files to be deployed (within the `--root`).  HTML files will be parsed, and the CSS/JS they point to will be included (versioned) automatically.  If you also include those files in your glob pattern they will be uploaded twice, once with a versioning hash in the URL, again without.
  
  Be sure to include any additional files you would like deployed like images, videos, font files, etc.

  You can use relative paths which break out of the `root`.  If you prefix the path with `-/`, it will be interpreted as relative to the project directory, not the `root`.
  	
##### `env`
  The config file can contain configurations for multiple environments (production, staging, etc.).  This specifies which is used.  See the "YAML Config" section for more information.

##### `key`
  The AWS key to use.  The create command will create an IAM user for each project with access only to the relevant bucket.  See the Permissions section for more information.
  
##### `secret`
  The AWS secret of the provided key.

##### `region` ("us-east-1")
  The AWS region the S3 bucket is located in.
   
### YAML Config

You can provide a yaml file which specifies configuration defaults for the project being deployed.  We include this file in each project which will be deployed.  This file can have multiple configurations for different environments, along with a default section.

For example, the `deploy.yaml` for one of our projects looks like:

```yaml
default:
  root: 'build/'

production:
  key: 'XXX'
  secret: 'XXX'
  bucket: 'eager.io'

dev:
  key: 'XXX'
  secret: 'XXX'
  bucket: 'next.eager.io'
```

Replacing the "XXX"s with our actual credentials.

To deploy to development we run (from the directory with the deploy.yaml file in it):

```bash
deploy --env development
```

A rollback of development would be:

```bash
rollback --env development $DEPLOY_ID
```

Where the deploy id is taken from the output of the deploy you wish to rollback to.
 
Our public projects use a similar config, but they specify the Amazon credentials as environment vars from the build system, passed in as flags:
 
 ```bash
deploy --env development --key $AMAZON_KEY_DEV --secret $AMAZON_SECRET_DEV
```

Never commit Amazon credentials to a file in a public repo.  Keep them on your local machine, or in your build system's configuration.

### Clean URLS

It's not specific to Stout, but it's worth mentioning that we recommend you structure your built folder to use a folder with an index.html file for each page.

For example, if you want a root and a page at `/blog`, you would have:

```
index.html
blog/
  index.html
```

That way, assuming S3 and CloudFront are configured properly, you'll be able to use the clean URLs `/` and `/blog/`.

### Permissions

The AWS user which is used for Stout should have the `GetObject`, `PutObject`, `DeleteObject`, and `ListBucket` permissions.  The `utils/create_site.sh` script will set this up for you if you use it to create your site.

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

### Deploying with CircleCI
 
 Deploying with CircleCI is simply a matter of installing the deploy tool and running it as you would locally.  Here's an excerpt of a working circle.yml:
 
 ```yaml
dependencies:
  post:
    - go get github.com/tools/godep
    - git clone git@github.com:EagerIO/Stout.git
    - cd Stout; godep go build -o ../stout src/*.go
    
deployment:
  development:
    branch: dev
    commands:
      - ./stout deploy --env development --key $AMAZON_KEY_DEV --secret $AMAZON_SECRET_DEV

  production:
    branch: master
    commands:
      - ./stout deploy --env production --key $AMAZON_KEY_PROD --secret $AMAZON_SECRET_PROD
 ```
 
If you use environment vars for your credentials, make sure to add them to your Circle config.
 
If your repo is private, you can specify your Amazon key and secret in your deploy.yaml file, removing the need to specify them in the commands.
 
### Caching

All versioned files (include a hash of their contents in the path) are configured to cache for one year.  All unversioned files are configured to cache for 60 seconds.  This means it will take up to 60 seconds for users to see changes made to your site.

### Versioning

Only JS and CSS files which are pointed to in HTML files are hashed, as we need to be able to update the HTML to point to our new, versioned, files.

Any other file included in your `--files` argument will be uploaded, but not versioned, meaning a rollback will not effect these files.  This is something we'd like to improve. 
 
### Consistency

As the final step of the deploy is atomic, multiple actors can trigger deploys simultaneously without any danger of inconsistent state.  Whichever process triggers the final 'copy' step for a given file will win, with it's specified dependencies guarenteed to be used in their entirity.  Note that this consistency is only guarenteed on a per-html-file level, you may end up with some html files from one deployer, and others from another, but all files will point to their correct dependencies.

### Deploying Multiple Projects To One Site

You can deploy multiple projects to the same domain simply by specifying the appropriate `dest` for each one.  For example your homepage might have the dest `./`, and your blog `./blog`.  Your homepage will be hosted at `your-site.com`, your blog `your-site.com/blog`.

### Using Client-side Routers

It is possible to use a client-side router (where you have multiple request URLs point to the same HTML file) by configuring your CloudFront distribution to serve your index.html file in response to 403s and 404s.

![CF](https://raw.githubusercontent.com/EagerIO/Stout/master/docs/images/cf-screenshot.png)

### Installing

- Download the release for your system type from our [releases](https://github.com/EagerIO/Stout/releases)
- Copy or symlink the `stout` binary contained in the archive into your path (for example, into `/usr/local/bin`)

### Building

- Install go and godep
- Run `godep restore ./...`
- Run `go build -o ../stout src/*`

#### For a Release (Cross Compiling)

- Run `go get github.com/laher/goxc`
- Run `go get code.google.com/p/go.tools/cmd/vet`
- Run `./utils/xc.sh`

The first run will take significantly longer than future runs.  The built files will be placed in the `./builds` directory.

### Running

To run the commands for development purposes, run: `go run src/*`, followed by any command line args you would normally give to the command.

### Contributing

Please do, we would love for this to become a project of the community.  Feel free to open an issue, submit a PR or contribute to the wiki.
