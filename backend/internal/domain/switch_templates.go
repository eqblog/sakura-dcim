package domain

// SwitchCommandTemplate defines a CLI command template for a specific operation.
type SwitchCommandTemplate struct {
	Operation   string `json:"operation"`   // e.g. "provision_access", "show_interface"
	Description string `json:"description"` // human-readable description
	Template    string `json:"template"`    // multiline CLI command template with {{placeholders}}
}

// SwitchVendorTemplates groups all templates for a single vendor.
type SwitchVendorTemplates struct {
	Vendor    string                  `json:"vendor"`     // canonical vendor key
	Label     string                  `json:"label"`      // display name
	Templates []SwitchCommandTemplate `json:"templates"`
}

// DefaultSwitchTemplates returns the built-in command templates for all supported vendors.
func DefaultSwitchTemplates() []SwitchVendorTemplates {
	return []SwitchVendorTemplates{
		ciscoIOSTemplates(),
		ciscoNXOSTemplates(),
		junosTemplates(),
		aristaEOSTemplates(),
		sonicTemplates(),
		cumulusTemplates(),
	}
}

func ciscoIOSTemplates() SwitchVendorTemplates {
	return SwitchVendorTemplates{
		Vendor: "cisco_ios",
		Label:  "Cisco IOS",
		Templates: []SwitchCommandTemplate{
			{
				Operation:   "provision_access",
				Description: "Configure access port with VLAN",
				Template: `configure terminal
interface {{port_name}}
description {{description}}
switchport mode access
switchport access vlan {{vlan_id}}
no shutdown
end
write memory`,
			},
			{
				Operation:   "provision_trunk",
				Description: "Configure trunk port with allowed VLANs",
				Template: `configure terminal
interface {{port_name}}
description {{description}}
switchport trunk encapsulation dot1q
switchport mode trunk
switchport trunk allowed vlan {{vlan_list}}
no shutdown
end
write memory`,
			},
			{
				Operation:   "vlan_create",
				Description: "Create a VLAN",
				Template: `configure terminal
vlan {{vlan_id}}
name {{vlan_name}}
end
write memory`,
			},
			{
				Operation:   "vlan_delete",
				Description: "Delete a VLAN",
				Template: `configure terminal
no vlan {{vlan_id}}
end
write memory`,
			},
			{
				Operation:   "port_channel",
				Description: "Add port to LACP port-channel",
				Template: `configure terminal
interface {{port_name}}
channel-group {{channel_id}} mode active
end
write memory`,
			},
			{
				Operation:   "port_shutdown",
				Description: "Shutdown a port",
				Template: `configure terminal
interface {{port_name}}
shutdown
end
write memory`,
			},
			{
				Operation:   "port_no_shutdown",
				Description: "Enable a port",
				Template: `configure terminal
interface {{port_name}}
no shutdown
end
write memory`,
			},
			{
				Operation:   "show_interface",
				Description: "Show interface status and counters",
				Template:    `show interface {{port_name}}`,
			},
			{
				Operation:   "show_vlan",
				Description: "Show VLAN information",
				Template:    `show vlan brief`,
			},
			{
				Operation:   "show_mac_table",
				Description: "Show MAC address table",
				Template:    `show mac address-table`,
			},
			{
				Operation:   "show_lldp",
				Description: "Show LLDP neighbor information",
				Template:    `show lldp neighbors`,
			},
			{
				Operation:   "show_running_config",
				Description: "Show running configuration for an interface",
				Template:    `show running-config interface {{port_name}}`,
			},
			{
				Operation:   "save_config",
				Description: "Save running config to startup",
				Template:    `write memory`,
			},
		},
	}
}

func ciscoNXOSTemplates() SwitchVendorTemplates {
	return SwitchVendorTemplates{
		Vendor: "cisco_nxos",
		Label:  "Cisco NX-OS (Nexus)",
		Templates: []SwitchCommandTemplate{
			{
				Operation:   "provision_access",
				Description: "Configure access port with VLAN",
				Template: `configure terminal
interface {{port_name}}
description {{description}}
switchport
switchport mode access
switchport access vlan {{vlan_id}}
no shutdown
end
copy running-config startup-config`,
			},
			{
				Operation:   "provision_trunk",
				Description: "Configure trunk port with allowed VLANs",
				Template: `configure terminal
interface {{port_name}}
description {{description}}
switchport
switchport mode trunk
switchport trunk allowed vlan {{vlan_list}}
no shutdown
end
copy running-config startup-config`,
			},
			{
				Operation:   "vlan_create",
				Description: "Create a VLAN",
				Template: `configure terminal
vlan {{vlan_id}}
name {{vlan_name}}
end
copy running-config startup-config`,
			},
			{
				Operation:   "vlan_delete",
				Description: "Delete a VLAN",
				Template: `configure terminal
no vlan {{vlan_id}}
end
copy running-config startup-config`,
			},
			{
				Operation:   "port_channel",
				Description: "Add port to LACP port-channel (vPC)",
				Template: `configure terminal
interface {{port_name}}
channel-group {{channel_id}} mode active
end
copy running-config startup-config`,
			},
			{
				Operation:   "vpc_config",
				Description: "Configure vPC peer link and keepalive",
				Template: `configure terminal
vpc domain {{vpc_domain_id}}
peer-keepalive destination {{peer_ip}}
interface port-channel {{channel_id}}
vpc peer-link
end
copy running-config startup-config`,
			},
			{
				Operation:   "port_shutdown",
				Description: "Shutdown a port",
				Template: `configure terminal
interface {{port_name}}
shutdown
end
copy running-config startup-config`,
			},
			{
				Operation:   "port_no_shutdown",
				Description: "Enable a port",
				Template: `configure terminal
interface {{port_name}}
no shutdown
end
copy running-config startup-config`,
			},
			{
				Operation:   "show_interface",
				Description: "Show interface status and counters",
				Template:    `show interface {{port_name}}`,
			},
			{
				Operation:   "show_interface_brief",
				Description: "Show all interfaces brief status",
				Template:    `show interface brief`,
			},
			{
				Operation:   "show_vlan",
				Description: "Show VLAN information",
				Template:    `show vlan brief`,
			},
			{
				Operation:   "show_mac_table",
				Description: "Show MAC address table",
				Template:    `show mac address-table`,
			},
			{
				Operation:   "show_lldp",
				Description: "Show LLDP neighbor information",
				Template:    `show lldp neighbors`,
			},
			{
				Operation:   "show_vpc",
				Description: "Show vPC status",
				Template:    `show vpc brief`,
			},
			{
				Operation:   "show_port_channel",
				Description: "Show port-channel summary",
				Template:    `show port-channel summary`,
			},
			{
				Operation:   "show_running_config",
				Description: "Show running configuration for an interface",
				Template:    `show running-config interface {{port_name}}`,
			},
			{
				Operation:   "save_config",
				Description: "Save running config to startup",
				Template:    `copy running-config startup-config`,
			},
		},
	}
}

func junosTemplates() SwitchVendorTemplates {
	return SwitchVendorTemplates{
		Vendor: "junos",
		Label:  "Juniper JunOS",
		Templates: []SwitchCommandTemplate{
			{
				Operation:   "provision_access",
				Description: "Configure access port with VLAN",
				Template: `set interfaces {{port_name}} description "{{description}}"
set interfaces {{port_name}} unit 0 family ethernet-switching interface-mode access
set interfaces {{port_name}} unit 0 family ethernet-switching vlan members vlan{{vlan_id}}
delete interfaces {{port_name}} disable
commit and-quit`,
			},
			{
				Operation:   "provision_trunk",
				Description: "Configure trunk port with allowed VLANs",
				Template: `set interfaces {{port_name}} description "{{description}}"
set interfaces {{port_name}} unit 0 family ethernet-switching interface-mode trunk
set interfaces {{port_name}} unit 0 family ethernet-switching vlan members [{{vlan_list}}]
delete interfaces {{port_name}} disable
commit and-quit`,
			},
			{
				Operation:   "vlan_create",
				Description: "Create a VLAN",
				Template: `set vlans vlan{{vlan_id}} vlan-id {{vlan_id}}
set vlans vlan{{vlan_id}} description "{{vlan_name}}"
commit and-quit`,
			},
			{
				Operation:   "vlan_delete",
				Description: "Delete a VLAN",
				Template: `delete vlans vlan{{vlan_id}}
commit and-quit`,
			},
			{
				Operation:   "port_channel",
				Description: "Add port to LACP aggregated ethernet (ae)",
				Template: `set interfaces {{port_name}} ether-options 802.3ad ae{{channel_id}}
commit and-quit`,
			},
			{
				Operation:   "port_shutdown",
				Description: "Disable a port",
				Template: `set interfaces {{port_name}} disable
commit and-quit`,
			},
			{
				Operation:   "port_no_shutdown",
				Description: "Enable a port",
				Template: `delete interfaces {{port_name}} disable
commit and-quit`,
			},
			{
				Operation:   "show_interface",
				Description: "Show interface status and counters",
				Template:    `show interfaces {{port_name}}`,
			},
			{
				Operation:   "show_vlan",
				Description: "Show VLAN information",
				Template:    `show vlans`,
			},
			{
				Operation:   "show_mac_table",
				Description: "Show MAC address table",
				Template:    `show ethernet-switching table`,
			},
			{
				Operation:   "show_lldp",
				Description: "Show LLDP neighbor information",
				Template:    `show lldp neighbors`,
			},
			{
				Operation:   "show_running_config",
				Description: "Show interface configuration",
				Template:    `show configuration interfaces {{port_name}}`,
			},
			{
				Operation:   "save_config",
				Description: "Commit configuration",
				Template:    `commit and-quit`,
			},
		},
	}
}

func aristaEOSTemplates() SwitchVendorTemplates {
	return SwitchVendorTemplates{
		Vendor: "arista_eos",
		Label:  "Arista EOS",
		Templates: []SwitchCommandTemplate{
			{
				Operation:   "provision_access",
				Description: "Configure access port with VLAN",
				Template: `configure
interface {{port_name}}
description {{description}}
switchport mode access
switchport access vlan {{vlan_id}}
no shutdown
end
write memory`,
			},
			{
				Operation:   "provision_trunk",
				Description: "Configure trunk port with allowed VLANs",
				Template: `configure
interface {{port_name}}
description {{description}}
switchport mode trunk
switchport trunk allowed vlan {{vlan_list}}
no shutdown
end
write memory`,
			},
			{
				Operation:   "vlan_create",
				Description: "Create a VLAN",
				Template: `configure
vlan {{vlan_id}}
name {{vlan_name}}
end
write memory`,
			},
			{
				Operation:   "vlan_delete",
				Description: "Delete a VLAN",
				Template: `configure
no vlan {{vlan_id}}
end
write memory`,
			},
			{
				Operation:   "port_channel",
				Description: "Add port to LACP port-channel",
				Template: `configure
interface {{port_name}}
channel-group {{channel_id}} mode active
end
write memory`,
			},
			{
				Operation:   "mlag_config",
				Description: "Configure MLAG domain",
				Template: `configure
mlag configuration
domain-id {{mlag_domain}}
local-interface Vlan4094
peer-address {{peer_ip}}
peer-link Port-Channel{{channel_id}}
end
write memory`,
			},
			{
				Operation:   "port_shutdown",
				Description: "Shutdown a port",
				Template: `configure
interface {{port_name}}
shutdown
end
write memory`,
			},
			{
				Operation:   "port_no_shutdown",
				Description: "Enable a port",
				Template: `configure
interface {{port_name}}
no shutdown
end
write memory`,
			},
			{
				Operation:   "show_interface",
				Description: "Show interface status and counters",
				Template:    `show interfaces {{port_name}}`,
			},
			{
				Operation:   "show_vlan",
				Description: "Show VLAN information",
				Template:    `show vlan brief`,
			},
			{
				Operation:   "show_mac_table",
				Description: "Show MAC address table",
				Template:    `show mac address-table`,
			},
			{
				Operation:   "show_lldp",
				Description: "Show LLDP neighbor information",
				Template:    `show lldp neighbors`,
			},
			{
				Operation:   "show_mlag",
				Description: "Show MLAG status",
				Template:    `show mlag`,
			},
			{
				Operation:   "show_running_config",
				Description: "Show running configuration for an interface",
				Template:    `show running-config interfaces {{port_name}}`,
			},
			{
				Operation:   "save_config",
				Description: "Save running config to startup",
				Template:    `write memory`,
			},
		},
	}
}

func sonicTemplates() SwitchVendorTemplates {
	return SwitchVendorTemplates{
		Vendor: "sonic",
		Label:  "SONiC",
		Templates: []SwitchCommandTemplate{
			{
				Operation:   "provision_access",
				Description: "Configure access port with VLAN",
				Template: `sudo config vlan member add {{vlan_id}} {{port_name}}
sudo config interface description {{port_name}} "{{description}}"
sudo config interface startup {{port_name}}
sudo config save -y`,
			},
			{
				Operation:   "provision_trunk",
				Description: "Configure trunk port (tagged VLAN member)",
				Template: `sudo config vlan member add {{vlan_id}} {{port_name}} --tagged
sudo config interface description {{port_name}} "{{description}}"
sudo config interface startup {{port_name}}
sudo config save -y`,
			},
			{
				Operation:   "vlan_create",
				Description: "Create a VLAN",
				Template: `sudo config vlan add {{vlan_id}}
sudo config save -y`,
			},
			{
				Operation:   "vlan_delete",
				Description: "Delete a VLAN",
				Template: `sudo config vlan del {{vlan_id}}
sudo config save -y`,
			},
			{
				Operation:   "port_channel",
				Description: "Add port to PortChannel",
				Template: `sudo config portchannel member add PortChannel{{channel_id}} {{port_name}}
sudo config save -y`,
			},
			{
				Operation:   "port_shutdown",
				Description: "Shutdown a port",
				Template: `sudo config interface shutdown {{port_name}}
sudo config save -y`,
			},
			{
				Operation:   "port_no_shutdown",
				Description: "Enable a port",
				Template: `sudo config interface startup {{port_name}}
sudo config save -y`,
			},
			{
				Operation:   "show_interface",
				Description: "Show interface status",
				Template:    `show interfaces status {{port_name}}`,
			},
			{
				Operation:   "show_vlan",
				Description: "Show VLAN information",
				Template:    `show vlan brief`,
			},
			{
				Operation:   "show_mac_table",
				Description: "Show MAC address table",
				Template:    `show mac`,
			},
			{
				Operation:   "show_lldp",
				Description: "Show LLDP neighbor information",
				Template:    `show lldp neighbors`,
			},
			{
				Operation:   "show_running_config",
				Description: "Show running configuration",
				Template:    `show runningconfiguration all`,
			},
			{
				Operation:   "save_config",
				Description: "Save configuration",
				Template:    `sudo config save -y`,
			},
		},
	}
}

func cumulusTemplates() SwitchVendorTemplates {
	return SwitchVendorTemplates{
		Vendor: "cumulus",
		Label:  "Cumulus Linux",
		Templates: []SwitchCommandTemplate{
			{
				Operation:   "provision_access",
				Description: "Configure access port with VLAN",
				Template: `net add bridge bridge ports {{port_name}}
net add interface {{port_name}} bridge access {{vlan_id}}
net add interface {{port_name}} alias "{{description}}"
net del interface {{port_name}} link down
net commit`,
			},
			{
				Operation:   "provision_trunk",
				Description: "Configure trunk port with allowed VLANs",
				Template: `net add bridge bridge ports {{port_name}}
net add interface {{port_name}} bridge trunk vlans {{vlan_list}}
net add interface {{port_name}} alias "{{description}}"
net del interface {{port_name}} link down
net commit`,
			},
			{
				Operation:   "vlan_create",
				Description: "Create a VLAN (bridge-aware)",
				Template: `net add bridge bridge vids {{vlan_id}}
net commit`,
			},
			{
				Operation:   "vlan_delete",
				Description: "Delete a VLAN",
				Template: `net del bridge bridge vids {{vlan_id}}
net commit`,
			},
			{
				Operation:   "port_channel",
				Description: "Add port to LACP bond",
				Template: `net add bond bond{{channel_id}} bond slaves {{port_name}}
net add bond bond{{channel_id}} bond mode 802.3ad
net commit`,
			},
			{
				Operation:   "clag_config",
				Description: "Configure CLAG (Multi-chassis LAG)",
				Template: `net add clag peer sys-mac {{sys_mac}}
net add clag peer interface {{peer_interface}} {{peer_ip}}/30
net add clag backup-ip {{backup_ip}}
net commit`,
			},
			{
				Operation:   "port_shutdown",
				Description: "Shutdown a port",
				Template: `net add interface {{port_name}} link down
net commit`,
			},
			{
				Operation:   "port_no_shutdown",
				Description: "Enable a port",
				Template: `net del interface {{port_name}} link down
net commit`,
			},
			{
				Operation:   "show_interface",
				Description: "Show interface status",
				Template:    `net show interface {{port_name}}`,
			},
			{
				Operation:   "show_vlan",
				Description: "Show bridge VLAN information",
				Template:    `net show bridge vlan`,
			},
			{
				Operation:   "show_mac_table",
				Description: "Show MAC address table",
				Template:    `net show bridge macs`,
			},
			{
				Operation:   "show_lldp",
				Description: "Show LLDP neighbor information",
				Template:    `net show lldp`,
			},
			{
				Operation:   "show_running_config",
				Description: "Show interface configuration",
				Template:    `net show configuration interface {{port_name}}`,
			},
			{
				Operation:   "save_config",
				Description: "Commit and apply configuration",
				Template:    `net commit`,
			},
		},
	}
}
