package amazon

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acm"
	"golang.org/x/net/publicsuffix"
)

// Find the best matching certificates to the domain name.
// return the best matching certificate if a certificate exists
// (see https://github.com/cloudflare/stout/issues/20#issuecomment-232174716)
// or, if there is no certificate with the domain requested, return ""
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

	// if --create-custom-ssl is specified, only look for exact match and stout tag, otherwise return nil and
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

// Check if the certificate is issued(confirmed)
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
	}

	return nil
}

// request a new certificate from ACM
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

// Set up ssl/tls certificates
func setUpSSL(awsSession *session.Session, domain string, createSSL bool) (string, error) {
	// if the person wants ssl certificates
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
	}

	if createSSL {
		fmt.Println("No certificate found to use, creating a new one.")
		validationEmails, err := requestCertificate(acmService, domain)
		if err != nil {
			return "", err
		}
		errorText := fmt.Sprintf("Please check one of the email addresses below to confirm your new SSL/TLS certificate and run this command again. \n\t- %s", strings.Join(validationEmails, "\n\t- "))
		return "", errors.New(errorText)
	}

	// set up without custom ssl
	return "", nil
}
