# cloudflare

The cloudflare provider supports DNS and DNS with CDN. This means that using cloudflare CDN requires using cloudflare DNS, but cloudflare DNS can be used with other CDN providers.

Go to [cloudflare.com](https://www.cloudflare.com/a/overview) to create an account or change settings on an existing install.

## Options

The `cf-email` and `cf-key` flags hold your account email and API key, respectively.

## Config

The providers section of an example config file using cloudflare could look like the following:

```yaml
[...]
        providers:
                cloudflare:
                        email: email@example.com
                        key: 4s2i3ax2y8e6dye13j20f9dx32
```
