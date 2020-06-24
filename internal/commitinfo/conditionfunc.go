package commitinfo

type conditionFunc = func(commitMsg string) bool

func LocateRelease(validator func(CommitInfo) bool) conditionFunc {
	return func(commitMsg string) bool {
		commitInfo, err := ParseCommitInfo(commitMsg)
		if err != nil {
			return false
		}
		return validator(commitInfo)
	}
}
