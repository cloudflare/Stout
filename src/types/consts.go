package types

type DNSRecordType string

const (
	CNAME_RECORD = DNSRecordType("CNAME")
	TXT_RECORD   = DNSRecordType("TXT")
)

type Provider string

const (
	DNS_PROVIDER = Provider("dns")
	FS_PROVIDER  = Provider("fs")
	CDN_PROVIDER = Provider("cdn")
)
