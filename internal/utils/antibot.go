package utils

import (
	"net/http"
	"regexp"
	"strings"
)

// Antibot detection ported from microlinkhq/is-antibot (MIT), limited to its
// "antibot" provider set. Each provider is matched on any of its detections:
// response headers, Set-Cookie prefixes, HTML markers, URL patterns or status.

type antibotHeaderRule struct {
	name        string
	equals      string
	startsWith  string
	oneOf       []string
	exists      bool
	except      string
	namePattern *regexp.Regexp
}

type antibotTextRule struct {
	contains string
	re       *regexp.Regexp
	all      []string
}

type antibotDetection struct {
	kind      string // headers, cookies, html, url
	technique string
	headers   []antibotHeaderRule
	texts     []antibotTextRule
	cookies   []string
}

type antibotProvider struct {
	name       string
	detections []antibotDetection
}

func eqHeader(name, value string) antibotHeaderRule {
	return antibotHeaderRule{name: name, equals: value}
}

func headerExists(name string) antibotHeaderRule {
	return antibotHeaderRule{name: name, exists: true}
}

func headerOneOf(name string, values ...string) antibotHeaderRule {
	return antibotHeaderRule{name: name, oneOf: values}
}

func headerStarts(name, prefix string) antibotHeaderRule {
	return antibotHeaderRule{name: name, startsWith: prefix}
}

func contains(text string) antibotTextRule {
	return antibotTextRule{contains: strings.ToLower(text)}
}

func matches(expr string) antibotTextRule {
	return antibotTextRule{re: regexp.MustCompile("(?is)" + expr)}
}

// allOf is used where upstream leans on lookaheads, which RE2 cannot do.
func allOf(values ...string) antibotTextRule {
	lowered := make([]string, len(values))
	for i, v := range values {
		lowered[i] = strings.ToLower(v)
	}
	return antibotTextRule{all: lowered}
}

func headers(detection antibotHeaderRule) antibotDetection {
	return antibotDetection{kind: "headers", technique: "javascript", headers: []antibotHeaderRule{detection}}
}

func htmlRules(technique string, rules ...antibotTextRule) antibotDetection {
	return antibotDetection{kind: "html", technique: technique, texts: rules}
}

func urlRules(rules ...antibotTextRule) antibotDetection {
	return antibotDetection{kind: "url", technique: "javascript", texts: rules}
}

func cookieRules(prefixes ...string) antibotDetection {
	return antibotDetection{kind: "cookies", technique: "cookie", cookies: prefixes}
}

var antibotProviders = []antibotProvider{
	{name: "cloudflare", detections: []antibotDetection{
		headers(eqHeader("cf-mitigated", "challenge")),
		cookieRules("cf_clearance="),
		htmlRules("javascript", contains("_cf_chl_opt")),
	}},
	{name: "vercel", detections: []antibotDetection{
		headers(eqHeader("x-vercel-mitigated", "challenge")),
	}},
	{name: "akamai", detections: []antibotDetection{
		{kind: "headers", technique: "waf", headers: []antibotHeaderRule{
			headerStarts("akamai-cache-status", "Error"),
			headerExists("x-akamai-session-info"),
		}},
		cookieRules("_abck="),
		htmlRules("javascript", contains("bmak.sensor_data")),
	}},
	{name: "datadome", detections: []antibotDetection{
		{kind: "headers", technique: "javascript", headers: []antibotHeaderRule{
			headerOneOf("x-dd-b", "1", "2"),
			{name: "x-datadome", exists: true, except: "protected"},
			headerExists("x-datadome-cid"),
		}},
		htmlRules("javascript", contains("captcha-delivery.com")),
		cookieRules("datadome="),
	}},
	{name: "perimeterx", detections: []antibotDetection{
		headers(headerExists("x-px-authorization")),
		htmlRules("javascript",
			allOf("window._pxAppId", "px-captcha"),
			allOf("window._pxAppId", "/captcha/captcha.js"),
			allOf("window._pxAppId", "_pxAction"),
			allOf("window._pxAppId", "robot or human"),
			contains("_pxAction"),
		),
		urlRules(contains("/blocked?")),
		cookieRules("_px3=", "_pxhd="),
	}},
	{name: "shapesecurity", detections: []antibotDetection{
		{kind: "headers", technique: "javascript", headers: []antibotHeaderRule{
			{namePattern: regexp.MustCompile(`^x-[a-z0-9]{8}-[abcdfz]$`)},
		}},
		htmlRules("javascript", matches(`shapesecurity\.\w+\s*\(`)),
	}},
	{name: "kasada", detections: []antibotDetection{
		{kind: "headers", technique: "javascript", headers: []antibotHeaderRule{
			headerExists("x-kasada"),
			headerExists("x-kasada-challenge"),
		}},
		htmlRules("javascript", contains("__kasada"), contains("kasada.js")),
	}},
	{name: "imperva", detections: []antibotDetection{
		htmlRules("javascript",
			matches(`_Incapsula_Resource\?[A-Za-z0-9_]+=[0-9a-f]{8,},[0-9a-f]{8,}`),
			matches(`(?:incapsula|imperva)\.\w+\s*\(`),
		),
		{kind: "headers", technique: "waf", headers: []antibotHeaderRule{
			eqHeader("x-cdn", "Incapsula"),
			headerExists("x-iinfo"),
		}},
		cookieRules("incap_ses_", "visid_incap_", "reese84="),
	}},
	{name: "reblaze", detections: []antibotDetection{
		cookieRules("rbzid=", "rbzsessionid="),
		htmlRules("waf", contains("Protected by Reblaze")),
	}},
	{name: "cheq", detections: []antibotDetection{
		htmlRules("javascript", contains("CheqSdk"), contains("cheqzone.com")),
		urlRules(matches(`cheqzone\.com`), matches(`cheq\.ai`)),
	}},
	{name: "sucuri", detections: []antibotDetection{
		htmlRules("waf", contains("Sucuri Website Firewall")),
	}},
	{name: "threatmetrix", detections: []antibotDetection{
		htmlRules("javascript", matches(`ThreatMetrix\.\w+\s*\(`)),
		urlRules(contains("fp/check.js")),
	}},
	{name: "meetrics", detections: []antibotDetection{
		htmlRules("javascript", matches(`meetricsGlobal\.\w+\s*\(`)),
		urlRules(matches(`meetrics\.com`)),
	}},
	{name: "ocule", detections: []antibotDetection{
		htmlRules("javascript", contains("ocule.co.uk")),
		urlRules(matches(`ocule\.co\.uk`)),
	}},
	{name: "anubis", detections: []antibotDetection{
		htmlRules("javascript",
			matches(`<script id="anubis_challenge"`),
			contains("/.within.website/x/cmd/anubis/"),
		),
	}},
	{name: "fullstory-challenge", detections: []antibotDetection{
		htmlRules("javascript", allOf("_fs-ch-", "client challenge")),
		cookieRules("_fs_ch_st_"),
	}},
	{name: "aws-waf", detections: []antibotDetection{
		headers(headerExists("x-amzn-waf-action")),
		htmlRules("javascript", matches(`aws-waf\.\w+\s*\(`), contains("gokuProps")),
		htmlRules("javascript", matches(`awswaf\.com|/awswaf/`)),
		cookieRules("aws-waf-token="),
	}},
	{name: "cloudfront", detections: []antibotDetection{
		{kind: "headers", technique: "waf", headers: []antibotHeaderRule{
			headerStarts("x-cache", "Error from cloudfront"),
		}},
		htmlRules("waf", contains("The request could not be satisfied")),
	}},
	{name: "hostinger", detections: []antibotDetection{
		htmlRules("javascript",
			contains("/hcdn-cgi/jschallenge"),
			contains("Checking your browser before accessing"),
		),
		headers(eqHeader("server", "hcdn")),
	}},
}

// BlockedByAntibot reports the provider that challenged the response, if any.
func BlockedByAntibot(status int, header http.Header, body, rawURL string) (string, bool) {
	if len(antibotProviders) == 0 {
		return "", false
	}

	lowerBody := strings.ToLower(body)
	lowerURL := strings.ToLower(rawURL)
	setCookies := strings.ToLower(strings.Join(header.Values("Set-Cookie"), "\n"))

	for _, provider := range antibotProviders {
		for _, detection := range provider.detections {
			if detection.matches(header, lowerBody, lowerURL, setCookies) {
				return provider.name, true
			}
		}
	}
	return "", false
}

func (d antibotDetection) matches(header http.Header, body, rawURL, setCookies string) bool {
	switch d.kind {
	case "headers":
		for _, rule := range d.headers {
			if rule.matches(header) {
				return true
			}
		}
	case "cookies":
		if setCookies == "" {
			return false
		}
		for _, prefix := range d.cookies {
			if strings.Contains(setCookies, strings.ToLower(prefix)) {
				return true
			}
		}
	case "html":
		for _, rule := range d.texts {
			if rule.matches(body) {
				return true
			}
		}
	case "url":
		for _, rule := range d.texts {
			if rule.matches(rawURL) {
				return true
			}
		}
	}
	return false
}

func (r antibotHeaderRule) matches(header http.Header) bool {
	if r.namePattern != nil {
		for name := range header {
			if r.namePattern.MatchString(strings.ToLower(name)) {
				return true
			}
		}
		return false
	}

	value := header.Get(r.name)
	if r.equals != "" {
		return strings.EqualFold(value, r.equals)
	}
	if r.startsWith != "" {
		return strings.HasPrefix(strings.ToLower(value), strings.ToLower(r.startsWith))
	}
	if len(r.oneOf) > 0 {
		for _, candidate := range r.oneOf {
			if strings.EqualFold(value, candidate) {
				return true
			}
		}
		return false
	}
	if !r.exists {
		return false
	}
	if value == "" {
		return false
	}
	return r.except == "" || !strings.EqualFold(value, r.except)
}

func (r antibotTextRule) matches(text string) bool {
	if r.re != nil {
		return r.re.MatchString(text)
	}
	if len(r.all) > 0 {
		for _, part := range r.all {
			if !strings.Contains(text, part) {
				return false
			}
		}
		return true
	}
	return strings.Contains(text, r.contains)
}
