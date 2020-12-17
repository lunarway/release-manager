package flux


func GetCommits(meta EventMetadata) []Commit {
	switch v := meta.(type) {
	case *CommitEventMetadata:
		return []Commit{
			{
				Revision: v.Revision,
			},
		}
	case *SyncEventMetadata:
		return v.Commits
	default:
		return []Commit{}
	}
}

func GetErrors(meta EventMetadata) []ResourceError {
	switch v := meta.(type) {
	case *SyncEventMetadata:
		return v.Errors
	default:
		return []ResourceError{}
	}
}
