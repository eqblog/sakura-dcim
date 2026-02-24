-- Seed: Default OS Profiles, Disk Layouts, and Post-Install Scripts

-- ===================
-- OS Profiles
-- ===================

-- Ubuntu 22.04 LTS (Autoinstall / cloud-init)
INSERT INTO os_profiles (id, name, os_family, version, arch, kernel_url, initrd_url, boot_args, template_type, template, is_active, tags) VALUES
('10000000-0000-0000-0000-000000000001', 'Ubuntu 22.04 LTS', 'ubuntu', '22.04', 'amd64',
 'http://archive.ubuntu.com/ubuntu/dists/jammy-updates/main/installer-amd64/current/legacy-images/netboot/ubuntu-installer/amd64/linux',
 'http://archive.ubuntu.com/ubuntu/dists/jammy-updates/main/installer-amd64/current/legacy-images/netboot/ubuntu-installer/amd64/initrd.gz',
 'auto=true priority=critical',
 'autoinstall',
 '#cloud-config
autoinstall:
  version: 1
  locale: en_US.UTF-8
  keyboard:
    layout: us
  identity:
    hostname: {{.Hostname}}
    password: {{.RootPasswordHash}}
    username: root
  ssh:
    install-server: true
    authorized-keys:
      {{range .SSHKeys}}- {{.}}
      {{end}}
  network:
    network:
      version: 2
      ethernets:
        ens3:
          addresses:
            - {{.IP}}/{{.Netmask}}
          gateway4: {{.Gateway}}
          nameservers:
            addresses: [8.8.8.8, 1.1.1.1]
  storage:
    layout:
      name: direct
  late-commands:
    - curtin in-target -- apt-get update
    - curtin in-target -- apt-get install -y openssh-server curl wget
',
 true, '{ubuntu,lts,22.04}');

-- Ubuntu 24.04 LTS
INSERT INTO os_profiles (id, name, os_family, version, arch, kernel_url, initrd_url, boot_args, template_type, template, is_active, tags) VALUES
('10000000-0000-0000-0000-000000000002', 'Ubuntu 24.04 LTS', 'ubuntu', '24.04', 'amd64',
 'http://archive.ubuntu.com/ubuntu/dists/noble/main/installer-amd64/current/legacy-images/netboot/ubuntu-installer/amd64/linux',
 'http://archive.ubuntu.com/ubuntu/dists/noble/main/installer-amd64/current/legacy-images/netboot/ubuntu-installer/amd64/initrd.gz',
 'auto=true priority=critical',
 'autoinstall',
 '#cloud-config
autoinstall:
  version: 1
  locale: en_US.UTF-8
  keyboard:
    layout: us
  identity:
    hostname: {{.Hostname}}
    password: {{.RootPasswordHash}}
    username: root
  ssh:
    install-server: true
    authorized-keys:
      {{range .SSHKeys}}- {{.}}
      {{end}}
  network:
    network:
      version: 2
      ethernets:
        ens3:
          addresses:
            - {{.IP}}/{{.Netmask}}
          gateway4: {{.Gateway}}
          nameservers:
            addresses: [8.8.8.8, 1.1.1.1]
  storage:
    layout:
      name: direct
',
 true, '{ubuntu,lts,24.04}');

-- Debian 12 (Preseed)
INSERT INTO os_profiles (id, name, os_family, version, arch, kernel_url, initrd_url, boot_args, template_type, template, is_active, tags) VALUES
('10000000-0000-0000-0000-000000000003', 'Debian 12 (Bookworm)', 'debian', '12', 'amd64',
 'http://deb.debian.org/debian/dists/bookworm/main/installer-amd64/current/images/netboot/debian-installer/amd64/linux',
 'http://deb.debian.org/debian/dists/bookworm/main/installer-amd64/current/images/netboot/debian-installer/amd64/initrd.gz',
 'auto=true priority=critical interface=auto',
 'preseed',
 'd-i debian-installer/locale string en_US.UTF-8
d-i keyboard-configuration/xkb-keymap select us
d-i netcfg/choose_interface select auto
d-i netcfg/get_hostname string {{.Hostname}}
d-i netcfg/get_domain string local

d-i mirror/country string manual
d-i mirror/http/hostname string deb.debian.org
d-i mirror/http/directory string /debian
d-i mirror/http/proxy string

d-i passwd/root-password-crypted password {{.RootPasswordHash}}
d-i passwd/make-user boolean false

d-i clock-setup/utc boolean true
d-i time/zone string UTC

d-i partman-auto/method string regular
d-i partman-auto/choose_recipe select atomic
d-i partman-partitioning/confirm_write_new_label boolean true
d-i partman/choose_partition select finish
d-i partman/confirm boolean true
d-i partman/confirm_nooverwrite boolean true

d-i apt-setup/non-free boolean true
d-i apt-setup/contrib boolean true
tasksel tasksel/first multiselect standard, ssh-server

d-i preseed/late_command string \
  in-target mkdir -p /root/.ssh; \
  {{range .SSHKeys}}echo "{{.}}" >> /target/root/.ssh/authorized_keys; \
  {{end}}in-target chmod 700 /root/.ssh; \
  in-target chmod 600 /root/.ssh/authorized_keys;

d-i finish-install/reboot_in_progress note
d-i grub-installer/only_debian boolean true
d-i grub-installer/bootdev string default
',
 true, '{debian,stable,12}');

-- CentOS Stream 9 (Kickstart)
INSERT INTO os_profiles (id, name, os_family, version, arch, kernel_url, initrd_url, boot_args, template_type, template, is_active, tags) VALUES
('10000000-0000-0000-0000-000000000004', 'CentOS Stream 9', 'centos', '9', 'amd64',
 'https://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/images/pxeboot/vmlinuz',
 'https://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/images/pxeboot/initrd.img',
 'inst.ks=http://{{.PanelIP}}:8080/api/v1/pxe/ks/{{.ServerID}}',
 'kickstart',
 '#version=RHEL9
install
url --url="https://mirror.stream.centos.org/9-stream/BaseOS/x86_64/os/"
lang en_US.UTF-8
keyboard us
timezone UTC --utc
rootpw --iscrypted {{.RootPasswordHash}}
network --bootproto=static --ip={{.IP}} --netmask={{.Netmask}} --gateway={{.Gateway}} --nameserver=8.8.8.8 --hostname={{.Hostname}} --activate
firewall --enabled --ssh
selinux --enforcing
bootloader --location=mbr
zerombr
clearpart --all --initlabel

autopart

{{range .SSHKeys}}sshkey --username=root "{{.}}"
{{end}}

%packages
@^minimal-environment
openssh-server
curl
wget
vim-enhanced
%end

%post
systemctl enable sshd
%end

reboot
',
 true, '{centos,stream,9,rhel}');

-- Rocky Linux 9 (Kickstart)
INSERT INTO os_profiles (id, name, os_family, version, arch, kernel_url, initrd_url, boot_args, template_type, template, is_active, tags) VALUES
('10000000-0000-0000-0000-000000000005', 'Rocky Linux 9', 'rocky', '9', 'amd64',
 'https://download.rockylinux.org/pub/rocky/9/BaseOS/x86_64/os/images/pxeboot/vmlinuz',
 'https://download.rockylinux.org/pub/rocky/9/BaseOS/x86_64/os/images/pxeboot/initrd.img',
 'inst.ks=http://{{.PanelIP}}:8080/api/v1/pxe/ks/{{.ServerID}}',
 'kickstart',
 '#version=RHEL9
install
url --url="https://download.rockylinux.org/pub/rocky/9/BaseOS/x86_64/os/"
lang en_US.UTF-8
keyboard us
timezone UTC --utc
rootpw --iscrypted {{.RootPasswordHash}}
network --bootproto=static --ip={{.IP}} --netmask={{.Netmask}} --gateway={{.Gateway}} --nameserver=8.8.8.8 --hostname={{.Hostname}} --activate
firewall --enabled --ssh
selinux --enforcing
bootloader --location=mbr
zerombr
clearpart --all --initlabel

autopart

{{range .SSHKeys}}sshkey --username=root "{{.}}"
{{end}}

%packages
@^minimal-environment
openssh-server
curl
wget
%end

%post
systemctl enable sshd
%end

reboot
',
 true, '{rocky,9,rhel}');

-- AlmaLinux 9 (Kickstart)
INSERT INTO os_profiles (id, name, os_family, version, arch, kernel_url, initrd_url, boot_args, template_type, template, is_active, tags) VALUES
('10000000-0000-0000-0000-000000000006', 'AlmaLinux 9', 'alma', '9', 'amd64',
 'https://repo.almalinux.org/almalinux/9/BaseOS/x86_64/os/images/pxeboot/vmlinuz',
 'https://repo.almalinux.org/almalinux/9/BaseOS/x86_64/os/images/pxeboot/initrd.img',
 'inst.ks=http://{{.PanelIP}}:8080/api/v1/pxe/ks/{{.ServerID}}',
 'kickstart',
 '#version=RHEL9
install
url --url="https://repo.almalinux.org/almalinux/9/BaseOS/x86_64/os/"
lang en_US.UTF-8
keyboard us
timezone UTC --utc
rootpw --iscrypted {{.RootPasswordHash}}
network --bootproto=static --ip={{.IP}} --netmask={{.Netmask}} --gateway={{.Gateway}} --nameserver=8.8.8.8 --hostname={{.Hostname}} --activate
firewall --enabled --ssh
selinux --enforcing
bootloader --location=mbr
zerombr
clearpart --all --initlabel

autopart

{{range .SSHKeys}}sshkey --username=root "{{.}}"
{{end}}

%packages
@^minimal-environment
openssh-server
curl
wget
%end

%post
systemctl enable sshd
%end

reboot
',
 true, '{alma,9,rhel}');

-- Proxmox VE 8 (answer file)
INSERT INTO os_profiles (id, name, os_family, version, arch, kernel_url, initrd_url, boot_args, template_type, template, is_active, tags) VALUES
('10000000-0000-0000-0000-000000000007', 'Proxmox VE 8', 'proxmox', '8', 'amd64',
 '', '',
 '',
 'cloud-init',
 '# Proxmox VE answer file
{
  "global": {
    "keyboard": "en-us",
    "country": "us",
    "fqdn": "{{.Hostname}}.local",
    "mailto": "admin@{{.Hostname}}.local",
    "timezone": "UTC",
    "root_password": "{{.RootPassword}}"
  },
  "network": {
    "source": "manual",
    "cidr": "{{.IP}}/{{.Netmask}}",
    "gateway": "{{.Gateway}}",
    "dns": "8.8.8.8"
  },
  "disk": {
    "type": "ext4",
    "disk": "sda"
  }
}
',
 true, '{proxmox,virtualization,8}');

-- Windows Server 2022 (Unattend.xml)
INSERT INTO os_profiles (id, name, os_family, version, arch, kernel_url, initrd_url, boot_args, template_type, template, is_active, tags) VALUES
('10000000-0000-0000-0000-000000000008', 'Windows Server 2022', 'windows', '2022', 'amd64',
 '', '',
 '',
 'cloud-init',
 '<?xml version="1.0" encoding="utf-8"?>
<unattend xmlns="urn:schemas-microsoft-com:unattend">
  <settings pass="specialize">
    <component name="Microsoft-Windows-Shell-Setup">
      <ComputerName>{{.Hostname}}</ComputerName>
      <TimeZone>UTC</TimeZone>
    </component>
    <component name="Microsoft-Windows-TCPIP">
      <Interfaces>
        <Interface wcm:action="add">
          <Identifier>Ethernet</Identifier>
          <UnicastIpAddresses>
            <IpAddress wcm:action="add" wcm:keyValue="1">{{.IP}}/{{.Netmask}}</IpAddress>
          </UnicastIpAddresses>
          <Routes>
            <Route wcm:action="add" wcm:keyValue="0">
              <NextHopAddress>{{.Gateway}}</NextHopAddress>
              <Prefix>0.0.0.0/0</Prefix>
            </Route>
          </Routes>
        </Interface>
      </Interfaces>
    </component>
  </settings>
  <settings pass="oobeSystem">
    <component name="Microsoft-Windows-Shell-Setup">
      <UserAccounts>
        <AdministratorPassword>
          <Value>{{.RootPassword}}</Value>
          <PlainText>true</PlainText>
        </AdministratorPassword>
      </UserAccounts>
      <OOBE>
        <SkipMachineOOBE>true</SkipMachineOOBE>
      </OOBE>
    </component>
  </settings>
</unattend>
',
 true, '{windows,server,2022}');


-- ===================
-- Disk Layouts
-- ===================

-- Single disk, simple layout (boot + swap + root)
INSERT INTO disk_layouts (id, name, description, layout, tags) VALUES
('20000000-0000-0000-0000-000000000001',
 'Standard (boot + swap + root)',
 'Simple layout for single disk: 512MB boot, 4GB swap, rest as root ext4',
 '{"disks": [{"device": "/dev/sda", "partitions": [{"mount": "/boot", "size": "512M", "fs": "ext4"}, {"mount": "swap", "size": "4G", "fs": "swap"}, {"mount": "/", "size": "max", "fs": "ext4"}]}]}',
 '{standard,single-disk}');

-- LVM layout
INSERT INTO disk_layouts (id, name, description, layout, tags) VALUES
('20000000-0000-0000-0000-000000000002',
 'LVM (boot + LVM root + swap)',
 'LVM-based layout: 512MB boot, LVM volume group with root and swap logical volumes',
 '{"disks": [{"device": "/dev/sda", "partitions": [{"mount": "/boot", "size": "512M", "fs": "ext4"}, {"mount": "pv", "size": "max", "fs": "lvm"}]}], "volume_groups": [{"name": "vg0", "pvs": ["/dev/sda2"], "lvs": [{"name": "root", "mount": "/", "size": "max", "fs": "ext4"}, {"name": "swap", "mount": "swap", "size": "4G", "fs": "swap"}]}]}',
 '{lvm,flexible}');

-- Separate /var partition (for servers with heavy logging/data)
INSERT INTO disk_layouts (id, name, description, layout, tags) VALUES
('20000000-0000-0000-0000-000000000003',
 'Separate /var (boot + root + var + swap)',
 'For database/logging servers: 512MB boot, 30GB root, rest for /var, 4GB swap',
 '{"disks": [{"device": "/dev/sda", "partitions": [{"mount": "/boot", "size": "512M", "fs": "ext4"}, {"mount": "swap", "size": "4G", "fs": "swap"}, {"mount": "/", "size": "30G", "fs": "ext4"}, {"mount": "/var", "size": "max", "fs": "ext4"}]}]}',
 '{separate-var,database,logging}');

-- Large /home partition (for shared hosting / user data)
INSERT INTO disk_layouts (id, name, description, layout, tags) VALUES
('20000000-0000-0000-0000-000000000004',
 'Separate /home (boot + root + home + swap)',
 'For user-facing servers: 512MB boot, 20GB root, rest for /home, 4GB swap',
 '{"disks": [{"device": "/dev/sda", "partitions": [{"mount": "/boot", "size": "512M", "fs": "ext4"}, {"mount": "swap", "size": "4G", "fs": "swap"}, {"mount": "/", "size": "20G", "fs": "ext4"}, {"mount": "/home", "size": "max", "fs": "ext4"}]}]}',
 '{separate-home,hosting}');

-- Full disk XFS (for high-performance storage)
INSERT INTO disk_layouts (id, name, description, layout, tags) VALUES
('20000000-0000-0000-0000-000000000005',
 'Full Disk XFS',
 'Entire disk as XFS with EFI boot: 512M EFI, 4G swap, rest as root XFS',
 '{"disks": [{"device": "/dev/sda", "partitions": [{"mount": "/boot/efi", "size": "512M", "fs": "vfat"}, {"mount": "swap", "size": "4G", "fs": "swap"}, {"mount": "/", "size": "max", "fs": "xfs"}]}]}',
 '{xfs,performance,efi}');

-- RAID1 boot + large data partition
INSERT INTO disk_layouts (id, name, description, layout, tags) VALUES
('20000000-0000-0000-0000-000000000006',
 'RAID1 Mirror (2 disks)',
 'Software RAID1 across 2 disks: mirrored boot, mirrored root, swap on first disk',
 '{"raid": {"level": "raid1", "devices": ["/dev/sda", "/dev/sdb"]}, "disks": [{"device": "/dev/md0", "partitions": [{"mount": "/boot", "size": "512M", "fs": "ext4"}, {"mount": "swap", "size": "4G", "fs": "swap"}, {"mount": "/", "size": "max", "fs": "ext4"}]}]}',
 '{raid1,mirror,redundant}');

-- Proxmox ZFS layout
INSERT INTO disk_layouts (id, name, description, layout, tags) VALUES
('20000000-0000-0000-0000-000000000007',
 'ZFS (Proxmox / FreeBSD)',
 'ZFS root pool on single disk, suitable for Proxmox VE and FreeBSD',
 '{"disks": [{"device": "/dev/sda", "partitions": [{"mount": "/boot/efi", "size": "512M", "fs": "vfat"}, {"mount": "/", "size": "max", "fs": "zfs", "pool": "rpool"}]}]}',
 '{zfs,proxmox,freebsd}');


-- ===================
-- Post-Install Scripts
-- ===================

-- Basic server hardening
INSERT INTO scripts (id, name, description, content, run_order, os_profile_ids, tags) VALUES
('30000000-0000-0000-0000-000000000001',
 'Basic Hardening',
 'Disable root password login, configure SSH, set timezone, update packages',
 '#!/bin/bash
set -e

# Disable root password SSH login (key-only)
sed -i "s/#PermitRootLogin.*/PermitRootLogin prohibit-password/" /etc/ssh/sshd_config
sed -i "s/PermitRootLogin yes/PermitRootLogin prohibit-password/" /etc/ssh/sshd_config
sed -i "s/#PasswordAuthentication.*/PasswordAuthentication no/" /etc/ssh/sshd_config
systemctl restart sshd

# Set timezone
timedatectl set-timezone UTC

# Update packages
if command -v apt-get &>/dev/null; then
    apt-get update && apt-get upgrade -y
elif command -v dnf &>/dev/null; then
    dnf update -y
elif command -v yum &>/dev/null; then
    yum update -y
fi

echo "[Sakura DCIM] Basic hardening complete"
',
 10,
 '{}',
 '{hardening,ssh,security}');

-- Install monitoring agent
INSERT INTO scripts (id, name, description, content, run_order, os_profile_ids, tags) VALUES
('30000000-0000-0000-0000-000000000002',
 'Install Node Exporter',
 'Install Prometheus node_exporter for server monitoring',
 '#!/bin/bash
set -e

NODE_EXPORTER_VERSION="1.7.0"
cd /tmp
curl -LO "https://github.com/prometheus/node_exporter/releases/download/v${NODE_EXPORTER_VERSION}/node_exporter-${NODE_EXPORTER_VERSION}.linux-amd64.tar.gz"
tar xzf "node_exporter-${NODE_EXPORTER_VERSION}.linux-amd64.tar.gz"
cp "node_exporter-${NODE_EXPORTER_VERSION}.linux-amd64/node_exporter" /usr/local/bin/
useradd --no-create-home --shell /bin/false node_exporter || true

cat > /etc/systemd/system/node_exporter.service <<EOF2
[Unit]
Description=Prometheus Node Exporter
After=network.target

[Service]
User=node_exporter
ExecStart=/usr/local/bin/node_exporter
Restart=always

[Install]
WantedBy=multi-user.target
EOF2

systemctl daemon-reload
systemctl enable --now node_exporter

echo "[Sakura DCIM] Node exporter installed on port 9100"
',
 20,
 '{}',
 '{monitoring,prometheus,node-exporter}');

-- Configure firewall
INSERT INTO scripts (id, name, description, content, run_order, os_profile_ids, tags) VALUES
('30000000-0000-0000-0000-000000000003',
 'Configure Firewall',
 'Set up firewall rules: allow SSH (22), HTTP (80), HTTPS (443), deny all other inbound',
 '#!/bin/bash
set -e

if command -v ufw &>/dev/null; then
    ufw default deny incoming
    ufw default allow outgoing
    ufw allow 22/tcp
    ufw allow 80/tcp
    ufw allow 443/tcp
    ufw --force enable
    echo "[Sakura DCIM] UFW firewall configured"
elif command -v firewall-cmd &>/dev/null; then
    firewall-cmd --permanent --add-service=ssh
    firewall-cmd --permanent --add-service=http
    firewall-cmd --permanent --add-service=https
    firewall-cmd --reload
    echo "[Sakura DCIM] firewalld configured"
else
    # Fallback: iptables
    iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
    iptables -A INPUT -p tcp --dport 22 -j ACCEPT
    iptables -A INPUT -p tcp --dport 80 -j ACCEPT
    iptables -A INPUT -p tcp --dport 443 -j ACCEPT
    iptables -A INPUT -i lo -j ACCEPT
    iptables -P INPUT DROP
    echo "[Sakura DCIM] iptables firewall configured"
fi
',
 30,
 '{}',
 '{firewall,security}');
