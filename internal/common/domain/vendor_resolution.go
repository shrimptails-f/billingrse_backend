package domain

import (
	"errors"
	"net/mail"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	// ErrVendorResolutionUnresolved は canonical Vendor が未解決のときに返る。
	ErrVendorResolutionUnresolved = errors.New("vendor resolution is unresolved")
)

const (
	// MatchedBy* はどのルールで Vendor を解決したかを表す監査用コード。
	MatchedByNameExact      = "name_exact"
	MatchedBySenderDomain   = "sender_domain"
	MatchedBySenderName     = "sender_name"
	MatchedBySubjectKeyword = "subject_keyword"
)

// VendorResolution は vendor 正規化の最終結果を表す。
// ParsedEmail の候補値ではなく、canonical Vendor を保持する。
type VendorResolution struct {
	ResolvedVendor *Vendor
}

// VendorResolutionInput は vendor 正規化に必要な生入力をまとめたもの。
type VendorResolutionInput struct {
	CandidateVendorName *string
	Subject             string
	From                string
	To                  []string
}

// VendorResolutionFetchPlan は repository が DB から集めるべき検索条件を表す。
type VendorResolutionFetchPlan struct {
	UserID            uint
	NameExactValue    string
	SenderDomainValue string
	SenderNameValue   string
	SubjectValue      string
}

// VendorAliasCandidate は alias lookup で得られた候補 1 件を表す。
type VendorAliasCandidate struct {
	AliasID         uint
	AliasType       string
	AliasValue      string
	NormalizedValue string
	AliasCreatedAt  time.Time
	Vendor          Vendor
}

// VendorResolutionFacts は repository が集めた判定材料をまとめたもの。
type VendorResolutionFacts struct {
	NameExactCandidates      []VendorAliasCandidate
	SenderDomainCandidates   []VendorAliasCandidate
	SenderNameCandidates     []VendorAliasCandidate
	SubjectKeywordCandidates []VendorAliasCandidate
}

// VendorRegistrationAlias は自動登録時に追加したい alias を表す。
type VendorRegistrationAlias struct {
	AliasType       string
	AliasValue      string
	NormalizedValue string
}

// VendorRegistrationPlan は unresolved 時に補完登録すべき内容を表す。
type VendorRegistrationPlan struct {
	UserID               uint
	VendorName           string
	NormalizedVendorName string
	Aliases              []VendorRegistrationAlias
}

// VendorResolutionDecision は 1 回の判定結果を表す。
type VendorResolutionDecision struct {
	Resolution VendorResolution
	MatchedBy  string
}

// VendorResolutionPolicy は vendor 解決ルールと登録候補生成ルールを司る。
type VendorResolutionPolicy struct{}

// IsResolved は canonical Vendor が解決済みかを返す。
func (r VendorResolution) IsResolved() bool {
	return r.ResolvedVendor != nil
}

// Validate は解決済み結果として成立しているかを検証する。
func (r VendorResolution) Validate() error {
	if !r.IsResolved() {
		return ErrVendorResolutionUnresolved
	}
	return r.ResolvedVendor.Validate()
}

// Normalize は自由入力文字列と宛先一覧を整形する。
func (in VendorResolutionInput) Normalize() VendorResolutionInput {
	in.CandidateVendorName = normalizeOptionalString(in.CandidateVendorName)
	in.Subject = strings.TrimSpace(in.Subject)
	in.From = strings.TrimSpace(in.From)
	in.To = normalizeStrings(in.To)
	return in
}

// BuildFetchPlan は生入力から repository が引くべき検索条件を構築する。
func (VendorResolutionPolicy) BuildFetchPlan(input VendorResolutionInput) VendorResolutionFetchPlan {
	input = input.Normalize()

	return VendorResolutionFetchPlan{
		NameExactValue:    NormalizeLooseText(stringValue(input.CandidateVendorName)),
		SenderDomainValue: extractSenderDomain(input.From),
		SenderNameValue:   NormalizeLooseText(extractSenderName(input.From)),
		SubjectValue:      NormalizeLooseText(input.Subject),
	}
}

// Resolve は repository が集めた材料に優先順位ルールを適用して最終判定する。
func (VendorResolutionPolicy) Resolve(facts VendorResolutionFacts) VendorResolutionDecision {
	if candidate := selectLatestAliasCandidate(facts.NameExactCandidates); candidate != nil {
		return resolvedDecision(candidate.Vendor, MatchedByNameExact)
	}
	if candidate := selectLatestAliasCandidate(facts.SenderDomainCandidates); candidate != nil {
		return resolvedDecision(candidate.Vendor, MatchedBySenderDomain)
	}
	if candidate := selectLatestAliasCandidate(facts.SenderNameCandidates); candidate != nil {
		return resolvedDecision(candidate.Vendor, MatchedBySenderName)
	}
	if vendor := resolveSubjectKeyword(facts.SubjectKeywordCandidates); vendor != nil {
		return resolvedDecision(*vendor, MatchedBySubjectKeyword)
	}

	return VendorResolutionDecision{}
}

// BuildRegistrationPlan は unresolved の candidate vendor 名から補完登録内容を作る。
func (VendorResolutionPolicy) BuildRegistrationPlan(input VendorResolutionInput, decision VendorResolutionDecision) *VendorRegistrationPlan {
	if decision.Resolution.IsResolved() {
		return nil
	}

	input = input.Normalize()
	candidateName := stringValue(input.CandidateVendorName)
	normalizedName := NormalizeLooseText(candidateName)
	if normalizedName == "" {
		return nil
	}

	return &VendorRegistrationPlan{
		VendorName:           candidateName,
		NormalizedVendorName: normalizedName,
		Aliases: []VendorRegistrationAlias{
			{
				AliasType:       MatchedByNameExact,
				AliasValue:      candidateName,
				NormalizedValue: normalizedName,
			},
		},
	}
}

// ResolveRegisteredVendor は自動登録で確定した Vendor を name_exact 解決結果へ変換する。
func (VendorResolutionPolicy) ResolveRegisteredVendor(vendor Vendor) VendorResolutionDecision {
	return resolvedDecision(vendor, MatchedByNameExact)
}

// NormalizeLooseText は大小文字・前後空白・連続空白の揺れを吸収する。
func NormalizeLooseText(value string) string {
	if value == "" {
		return ""
	}
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}

func resolvedDecision(vendor Vendor, matchedBy string) VendorResolutionDecision {
	return VendorResolutionDecision{
		Resolution: VendorResolution{
			ResolvedVendor: &vendor,
		},
		MatchedBy: matchedBy,
	}
}

func selectLatestAliasCandidate(candidates []VendorAliasCandidate) *VendorAliasCandidate {
	if len(candidates) == 0 {
		return nil
	}

	best := candidates[0]
	for _, candidate := range candidates[1:] {
		if newerAliasCandidate(candidate, best) {
			best = candidate
		}
	}
	return &best
}

func resolveSubjectKeyword(candidates []VendorAliasCandidate) *Vendor {
	if len(candidates) == 0 {
		return nil
	}

	sorted := append([]VendorAliasCandidate(nil), candidates...)
	sort.SliceStable(sorted, func(i, j int) bool {
		lenI := utf8.RuneCountInString(sorted[i].NormalizedValue)
		lenJ := utf8.RuneCountInString(sorted[j].NormalizedValue)
		if lenI != lenJ {
			return lenI > lenJ
		}
		if !sorted[i].AliasCreatedAt.Equal(sorted[j].AliasCreatedAt) {
			return sorted[i].AliasCreatedAt.After(sorted[j].AliasCreatedAt)
		}
		return sorted[i].AliasID > sorted[j].AliasID
	})

	longest := utf8.RuneCountInString(sorted[0].NormalizedValue)
	vendorIDs := make(map[uint]struct{})
	selected := sorted[0]

	for _, candidate := range sorted {
		if utf8.RuneCountInString(candidate.NormalizedValue) != longest {
			break
		}
		vendorIDs[candidate.Vendor.ID] = struct{}{}
		if newerAliasCandidate(candidate, selected) {
			selected = candidate
		}
	}

	if len(vendorIDs) > 1 {
		return nil
	}

	vendor := selected.Vendor
	return &vendor
}

func newerAliasCandidate(left, right VendorAliasCandidate) bool {
	if !left.AliasCreatedAt.Equal(right.AliasCreatedAt) {
		return left.AliasCreatedAt.After(right.AliasCreatedAt)
	}
	return left.AliasID > right.AliasID
}

func extractSenderDomain(rawFrom string) string {
	address := extractSenderAddress(rawFrom)
	if address == "" {
		return ""
	}

	at := strings.LastIndex(address, "@")
	if at < 0 || at == len(address)-1 {
		return ""
	}

	return strings.ToLower(strings.TrimSpace(address[at+1:]))
}

func extractSenderName(rawFrom string) string {
	parsed := strings.TrimSpace(rawFrom)
	if parsed == "" {
		return ""
	}

	if addr, err := mail.ParseAddress(parsed); err == nil {
		return strings.TrimSpace(addr.Name)
	}
	if idx := strings.Index(parsed, "<"); idx > 0 {
		return strings.TrimSpace(parsed[:idx])
	}

	return ""
}

func extractSenderAddress(rawFrom string) string {
	parsed := strings.TrimSpace(rawFrom)
	if parsed == "" {
		return ""
	}

	if addr, err := mail.ParseAddress(parsed); err == nil {
		return strings.ToLower(strings.TrimSpace(addr.Address))
	}
	if start := strings.Index(parsed, "<"); start >= 0 {
		if end := strings.Index(parsed[start+1:], ">"); end >= 0 {
			return strings.ToLower(strings.TrimSpace(parsed[start+1 : start+1+end]))
		}
	}
	if strings.Contains(parsed, "@") {
		return strings.ToLower(parsed)
	}
	return ""
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func normalizeStrings(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	return normalized
}
