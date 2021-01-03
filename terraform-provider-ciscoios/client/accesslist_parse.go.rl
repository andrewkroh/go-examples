package client

import (
    "fmt"
)

%%{
machine accesslist;

action mark {
    pb = p
}

action set_id {
    ale.ID = string(data[pb:p])
}

action set_remark {
    ale.Remark = string(data[pb:p])
}

action set_permit {
    if string(data[pb:p]) == "permit" {
        ale.Permit = true
    }
}

action set_protocol {
    ale.Protocol = string(data[pb:p])
}

action set_src_ip {
    ale.Source = string(data[pb:p])
}

action set_src_wildcard {
    ale.SourceWildcard = string(data[pb:p])
}

action set_src_port {
    ale.SourcePort = string(data[pb:p])
}

action set_dst_ip {
    ale.Destination = string(data[pb:p])
}

action set_dst_wildcard {
    ale.DestinationWildcard = string(data[pb:p])
}

action set_dst_port {
    ale.DestinationPort = string(data[pb:p])
}

action set_icmp_type {
    ale.ICMPType = string(data[pb:p])
}

action set_icmp_code {
    ale.ICMPCode = string(data[pb:p])
}

action set_igmp_type {
    ale.IGMPType = string(data[pb:p])
}

action set_established {
    ale.Established = true
}

action set_log {
    ale.Log = true
}

# whitespace
sp = ' ';
word = [a-z]+ ('-' [a-z]+);

port = digit+ | [a-z]+;
port_number = ('eq' | 'gt' | 'lt' | 'neq') sp port;
port_range = 'range' sp port sp port;

id = digit+ >mark %set_id;

permit_deny = ('permit' | 'deny') >mark %set_permit;

protocol = [a-z0-9]+ >mark %set_protocol;
tcp_protocol = 'tcp' >mark %set_protocol;
udp_protocol = 'udp' >mark %set_protocol;
icmp_protocol = 'icmp' >mark %set_protocol;
igmp_protocol = 'igmp' >mark %set_protocol;

host = 'host';
any_addr = 'any';
ip = digit{1,3} '.' digit{1,3} '.' digit{1,3} '.' digit{1,3};
log = sp 'log' >mark %set_log;
established = sp 'established' >mark %set_established;

src_ip = ip >mark %set_src_ip;
src_any = any_addr >mark %set_src_ip;
src_wildcard = ip >mark %set_src_wildcard;
src_port = (port_number | port_range) >mark %set_src_port;
src_addr = (host sp src_ip | src_any | src_ip sp src_wildcard);

dst_ip = ip >mark %set_dst_ip;
dst_any = any_addr >mark %set_dst_ip;
dst_wildcard = ip >mark %set_dst_wildcard;
dst_port = (port_number | port_range) >mark %set_dst_port;
dst_addr = (host sp dst_ip | dst_any | dst_ip sp dst_wildcard);

icmp_type = digit+ >mark %set_icmp_type;
icmp_code = digit+ >mark %set_icmp_code;
icmp_message = word >mark %set_icmp_type;
icmp_meta = (icmp_type | icmp_type sp icmp_code | icmp_message);

igmp_type = (word | digit+) >mark %set_igmp_type;

remark_msg = any+ >mark %set_remark;
remark = 'remark' sp remark_msg;

tcp_rule = tcp_protocol sp src_addr (sp src_port)? sp dst_addr (sp dst_port)? established?;
udp_rule = udp_protocol sp src_addr (sp src_port)? sp dst_addr (sp dst_port)?;
icmp_rule = icmp_protocol sp src_addr sp dst_addr sp icmp_meta;
igmp_rule = igmp_protocol sp src_addr sp dst_addr sp igmp_type;
ip_rule = protocol sp src_addr sp dst_addr;

rule = permit_deny sp+ (tcp_rule | udp_rule | icmp_rule | igmp_rule | ip_rule) log?;

main := 'access-list' sp id sp (remark | rule);

write data noerror;
}%%

func (ale *AccessListEntry) Parse(data string) error {
    cs, p, pb, pe, eof := 0, 0, 0, len(data), len(data)

	%% write init;
	%% write exec;

//    if cs < acl_first_final {
//		return fmt.Errorf("parsing failed")
//	}

    _ = eof

	return nil
}

