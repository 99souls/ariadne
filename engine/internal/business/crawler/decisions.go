package crawler

import (
	"context"
	"net/url"
	"time"
)

// CrawlDecision represents a decision about whether to crawl a URL
type CrawlDecision struct {
	URL              *url.URL
	ShouldCrawl      bool
	Reason           string
	ContentSelectors []string
	RequestDelay     time.Duration
	Domain           string
	CurrentDepth     int
}

// LinkDecision represents a decision about whether to follow a link
type LinkDecision struct {
	BaseURL      *url.URL
	LinkURL      *url.URL
	ShouldFollow bool
	Reason       string
	NextDepth    int
}

// CrawlingContextKey is a key type for crawling context values
type CrawlingContextKey string

// CrawlingDecisionMaker makes business decisions about crawling operations
type CrawlingDecisionMaker struct {
	evaluator *CrawlingPolicyEvaluator
}

// NewCrawlingDecisionMaker creates a new decision maker
func NewCrawlingDecisionMaker(policy *CrawlingBusinessPolicy) *CrawlingDecisionMaker {
	return &CrawlingDecisionMaker{
		evaluator: NewCrawlingPolicyEvaluator(policy),
	}
}

// ShouldCrawl makes a decision about whether a URL should be crawled
func (dm *CrawlingDecisionMaker) ShouldCrawl(ctx context.Context, u *url.URL, currentDepth int) (*CrawlDecision, error) {
	decision := &CrawlDecision{
		URL:          u,
		Domain:       u.Host,
		CurrentDepth: currentDepth,
	}

	// Check if URL is allowed
	if !dm.evaluator.IsURLAllowed(u) {
		decision.ShouldCrawl = false
		decision.Reason = "domain not allowed"
		return decision, nil
	}

	// Check if depth is within limits
	if !dm.evaluator.ShouldFollowLink(u, currentDepth) {
		decision.ShouldCrawl = false
		decision.Reason = "exceeds maximum depth"
		return decision, nil
	}

	// If all checks pass, prepare crawling parameters
	decision.ShouldCrawl = true
	decision.ContentSelectors = dm.evaluator.GetContentSelectors(u)
	decision.RequestDelay = dm.evaluator.GetRequestDelay(u.Host)

	return decision, nil
}

// ShouldFollowLink makes a decision about whether to follow a discovered link
func (dm *CrawlingDecisionMaker) ShouldFollowLink(ctx context.Context, baseURL, linkURL *url.URL, currentDepth int) (*LinkDecision, error) {
	decision := &LinkDecision{
		BaseURL:   baseURL,
		LinkURL:   linkURL,
		NextDepth: currentDepth + 1,
	}

	// First check depth limits
	if currentDepth+1 >= dm.evaluator.policy.LinkRules.MaxDepth {
		decision.ShouldFollow = false
		decision.Reason = "exceeds maximum depth"
		return decision, nil
	}

	// Check if this is an external link
	isExternal := baseURL.Host != linkURL.Host

	// If it's external and external following is disabled
	if isExternal && !dm.evaluator.policy.LinkRules.FollowExternal {
		decision.ShouldFollow = false
		decision.Reason = "external links disabled"
		return decision, nil
	}

	// Check if the link URL domain is allowed (for both internal and external links)
	if !dm.evaluator.IsURLAllowed(linkURL) {
		decision.ShouldFollow = false
		decision.Reason = "target domain not allowed"
		return decision, nil
	}

	decision.ShouldFollow = true
	return decision, nil
}

// BatchShouldCrawl makes crawling decisions for multiple URLs
func (dm *CrawlingDecisionMaker) BatchShouldCrawl(ctx context.Context, urls []string, currentDepth int) ([]*CrawlDecision, error) {
	decisions := make([]*CrawlDecision, 0, len(urls))

	for _, urlStr := range urls {
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			// Add a decision with error
			decision := &CrawlDecision{
				ShouldCrawl: false,
				Reason:      "invalid URL: " + err.Error(),
			}
			decisions = append(decisions, decision)
			continue
		}

		decision, err := dm.ShouldCrawl(ctx, parsedURL, currentDepth)
		if err != nil {
			decision = &CrawlDecision{
				URL:         parsedURL,
				ShouldCrawl: false,
				Reason:      "decision error: " + err.Error(),
			}
		}
		decisions = append(decisions, decision)
	}

	return decisions, nil
}

// CreateCrawlingContext creates a context with crawling-specific information
func (dm *CrawlingDecisionMaker) CreateCrawlingContext(parent context.Context, u *url.URL) context.Context {
	ctx := context.WithValue(parent, CrawlingContextKey("domain"), u.Host)
	ctx = context.WithValue(ctx, CrawlingContextKey("url"), u.String())
	return ctx
}
