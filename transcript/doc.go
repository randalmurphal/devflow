// Package transcript provides recording and management of AI conversation transcripts.
//
// Core types:
//   - Transcript: A recorded conversation with metadata and turns
//   - Turn: A single message in a conversation (user, assistant, or tool)
//   - Manager: Interface for transcript lifecycle management
//   - FileStore: File-based transcript storage implementation
//   - Searcher: Grep-based transcript search
//   - Viewer: Transcript display and export
//
// Example usage:
//
//	store := transcript.NewFileStore(transcript.StoreConfig{
//	    BaseDir: ".devflow/runs",
//	})
//	err := store.StartRun("run-123", transcript.RunMetadata{
//	    FlowID: "ticket-to-pr",
//	})
//	err = store.RecordTurn("run-123", transcript.Turn{
//	    Role:    "assistant",
//	    Content: "I'll implement this feature...",
//	})
package transcript
