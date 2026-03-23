namespace go v1

struct AnalyzeTrafficRequest {
  1: required string text_input
}

struct AnalyzeTrafficResponse {
  1: required string text_output
}

struct PromptAnalysisRequest {
  1: required string prompt
}

struct PromptAnalysisSession {
  1: required string session_id
  2: required bool done
  3: optional string error_text
}

struct PromptChunkRequest {
  1: required string session_id
  2: optional i32 max_chunks
}

struct PromptChunkResponse {
  1: required list<string> chunks
  2: required bool done
  3: optional string error_text
}

struct PromptCancelRequest {
  1: required string session_id
}

struct PromptCancelResponse {
  1: required bool canceled
}

service AIService {
  AnalyzeTrafficResponse AnalyzeTraffic(1: AnalyzeTrafficRequest req)
  PromptAnalysisSession StartPromptAnalysis(1: PromptAnalysisRequest req)
  PromptChunkResponse ReadPromptChunks(1: PromptChunkRequest req)
  PromptCancelResponse CancelPromptAnalysis(1: PromptCancelRequest req)
}
