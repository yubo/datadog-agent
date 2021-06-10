#
# Cookbook Name:: dd-agent-disable-system-repos
# Recipe:: default
#
# Copyright (C) 2021-present Datadog
#
# All rights reserved - Do Not Redistribute
#

if ['redhat', 'centos', 'fedora'].include?(node[:platform])
  execute 'disable all yum repositories' do
    command 'yum-config-manager --disable "*"'
  end
elsif ['suse', 'opensuseleap'].include?(node[:platform])
  execute 'disable all zypper repositories' do
    command 'zypper mr -da'
  end
elsif ['ubuntu', 'debian'].include?(node[:platform])
  execute 'disable all apt repositories' do
    command 'mv /etc/apt/sources.list /etc/apt/sources.list.disabled'
  end
end
