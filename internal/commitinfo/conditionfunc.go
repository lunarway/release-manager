package commitinfo

import (
	"fmt"
	"regexp"
)

type conditionFunc = func(commitMsg string) bool

func LocateReleaseCondition(artifactID string) conditionFunc {
	r := regexp.MustCompile(fmt.Sprintf(`(?i)release %s($|\r\n|\r|\n|\sby\s\S+)`, regexp.QuoteMeta(artifactID)))
	return func(commitMsg string) bool {
		if artifactID == "" {
			return false
		}
		return r.MatchString(commitMsg)
	}
}

func LocateServiceReleaseRollbackSkipCondition(env, service string, n uint) conditionFunc {
	return func(commitMsg string) bool {
		releaseOK := LocateServiceReleaseCondition(env, service)(commitMsg)
		rollbackOK := LocateServiceRollbackCondition(env, service)(commitMsg)
		ok := releaseOK || rollbackOK
		if !ok {
			return false
		}
		if n == 0 {
			return true
		}
		n--
		return false
	}
}

func LocateServiceRollbackCondition(env, service string) conditionFunc {
	r := regexp.MustCompile(fmt.Sprintf(`(?i)\[%s/%s] rollback `, regexp.QuoteMeta(env), regexp.QuoteMeta(service)))
	return func(commitMsg string) bool {
		if env == "" {
			return false
		}
		if service == "" {
			return false
		}
		return r.MatchString(commitMsg)
	}
}

func LocateEnvReleaseCondition(env, artifactId string) conditionFunc {
	r := regexp.MustCompile(fmt.Sprintf(`(?i)\[%s/.*] release %s($|\r\n|\r|\n|\sby\s\S+)`, regexp.QuoteMeta(env), regexp.QuoteMeta(artifactId)))
	return func(commitMsg string) bool {
		if env == "" {
			return false
		}
		if artifactId == "" {
			return false
		}
		return r.MatchString(commitMsg)
	}
}

func LocateServiceReleaseCondition(env, service string) conditionFunc {
	r := regexp.MustCompile(fmt.Sprintf(`(?i)\[%s/%s] release`, regexp.QuoteMeta(env), regexp.QuoteMeta(service)))
	return func(commitMsg string) bool {
		if env == "" {
			return false
		}
		if service == "" {
			return false
		}
		return r.MatchString(commitMsg)
	}
}
