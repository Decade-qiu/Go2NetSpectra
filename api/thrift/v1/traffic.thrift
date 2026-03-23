namespace go v1

struct FiveTuple {
  1: required binary src_ip
  2: required binary dst_ip
  3: required i32 src_port
  4: required i32 dst_port
  5: required i32 protocol
}

struct PacketInfo {
  1: required i64 timestamp_unix_nano
  2: required FiveTuple five_tuple
  3: required i64 length
}
