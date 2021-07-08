#
# Cookbook Name:: dd-agent-debian-workaround
# Recipe:: default
#
# Copyright (C) 2021-present Datadog
#
# All rights reserved - Do Not Redistribute
#

include_recipe "hostsfile"

if node['platform_family'] == 'debian'
  hostsfile_entry '127.0.0.1' do
    hostname  node['hostname']
    aliases   ['localhost']
    action    :append
  end

  package 'install-dirmngr' do
    package_name 'dirmngr'
    action :install
    ignore_failure true # Can fail on older distros where the dirmngr package does not exist, but shouldn't prevent install.
  end
end
