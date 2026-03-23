namespace go v1

struct AggregationRequest {
  1: optional i64 end_time_unix_nano
  2: optional string task_name
  3: optional string src_ip
  4: optional string dst_ip
  5: optional i32 src_port
  6: optional i32 dst_port
  7: optional i32 protocol
}

struct TaskSummary {
  1: required string task_name
  2: required i64 total_bytes
  3: required i64 total_packets
  4: required i64 flow_count
}

struct QueryTotalCountsResponse {
  1: required list<TaskSummary> summaries
}

struct TraceFlowRequest {
  1: required string task_name
  2: required map<string, string> flow_keys
  3: optional i64 end_time_unix_nano
}

struct FlowLifecycle {
  1: required i64 first_seen_unix_nano
  2: required i64 last_seen_unix_nano
  3: required i64 total_packets
  4: required i64 total_bytes
}

struct TraceFlowResponse {
  1: required FlowLifecycle lifecycle
}

struct HealthCheckRequest {}

struct HealthCheckResponse {
  1: required string status
}

struct SearchTasksRequest {}

struct SearchTasksResponse {
  1: required list<string> task_names
}

struct HeavyHittersRequest {
  1: required string task_name
  2: required i32 type
  3: optional i64 end_time_unix_nano
  4: required i32 limit
}

struct HeavyHitter {
  1: required string flow
  2: required i64 value
}

struct HeavyHittersResponse {
  1: required list<HeavyHitter> hitters
}

service QueryService {
  HealthCheckResponse HealthCheck(1: HealthCheckRequest req)
  SearchTasksResponse SearchTasks(1: SearchTasksRequest req)
  QueryTotalCountsResponse AggregateFlows(1: AggregationRequest req)
  TraceFlowResponse TraceFlow(1: TraceFlowRequest req)
  HeavyHittersResponse QueryHeavyHitters(1: HeavyHittersRequest req)
}
