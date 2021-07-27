#
# Cookbook Name:: dd-agent-step-by-step
# Recipe:: default
#
# Copyright (C) 2013-present Datadog
#
# All rights reserved - Do Not Redistribute
#

case node['platform_family']
when 'debian'
  execute 'install dependencies' do
    command <<-EOF
      sudo apt-get update
      sudo apt-get install -y apt-transport-https curl gnupg
    EOF
  end

  execute 'install debian' do
    command <<-EOF
      sudo sh -c "echo \'deb [signed-by=/usr/share/keyrings/datadog-archive-keyring.gpg] #{node['dd-agent-step-by-step']['aptrepo']} #{node['dd-agent-step-by-step']['aptrepo_dist']} #{node['dd-agent-step-by-step']['agent_major_version']}\' > /etc/apt/sources.list.d/datadog.list"
      sudo touch /usr/share/keyrings/datadog-archive-keyring.gpg
      sudo chmod a+r /usr/share/keyrings/datadog-archive-keyring.gpg

      curl https://keys.datadoghq.com/DATADOG_APT_KEY_CURRENT.public | sudo gpg --no-default-keyring --keyring /usr/share/keyrings/datadog-archive-keyring.gpg --import --batch
      curl https://keys.datadoghq.com/DATADOG_APT_KEY_382E94DE.public | sudo gpg --no-default-keyring --keyring /usr/share/keyrings/datadog-archive-keyring.gpg --import --batch
      curl https://keys.datadoghq.com/DATADOG_APT_KEY_F14F620E.public | sudo gpg --no-default-keyring --keyring /usr/share/keyrings/datadog-archive-keyring.gpg --import --batch

      sudo cp -a /usr/share/keyrings/datadog-archive-keyring.gpg /etc/apt/trusted.gpg.d/

      sudo apt-get update
      sudo apt-get install #{node['dd-agent-step-by-step']['package_name']} -y -q
    EOF
  end

when 'rhel'
  protocol = node['platform_version'].to_i < 6 ? 'http' : 'https'

  file '/etc/yum.repos.d/datadog.repo' do
    content <<-EOF.gsub(/^ {6}/, '')
      [datadog]
      name = Datadog, Inc.
      baseurl = #{node['dd-agent-step-by-step']['yumrepo']}
      enabled=1
      gpgcheck=1
      repo_gpgcheck=1
      gpgkey=#{protocol}://keys.datadoghq.com/DATADOG_RPM_KEY_CURRENT.public
             #{protocol}://keys.datadoghq.com/DATADOG_RPM_KEY_FD4BF915.public
             #{protocol}://keys.datadoghq.com/DATADOG_RPM_KEY_E09422B3.public
    EOF
  end

  execute 'install rhel' do
    command <<-EOF
      sudo yum makecache
      sudo yum install -y #{node['dd-agent-step-by-step']['package_name']}
    EOF
  end

when 'suse'
  file '/etc/zypp/repos.d/datadog.repo' do
    content <<-EOF.gsub(/^ {6}/, '')
      [datadog]
      name=Datadog, Inc.
      enabled=1
      baseurl = #{node['dd-agent-step-by-step']['yumrepo_suse']}
      type=rpm-md
      gpgcheck=1
      repo_gpgcheck=1
      gpgkey=https://keys.datadoghq.com/DATADOG_RPM_KEY_CURRENT.public
             https://keys.datadoghq.com/DATADOG_RPM_KEY_FD4BF915.public
             https://keys.datadoghq.com/DATADOG_RPM_KEY_E09422B3.public
    EOF
  end

  execute 'install suse' do
    command <<-EOF
      sudo zypper --non-interactive --no-gpg-checks refresh datadog
      sudo rpm --import https://keys.datadoghq.com/DATADOG_RPM_KEY_CURRENT.public
      sudo rpm --import https://keys.datadoghq.com/DATADOG_RPM_KEY_FD4BF915.public
      sudo rpm --import https://keys.datadoghq.com/DATADOG_RPM_KEY_E09422B3.public
      sudo rpm --import https://keys.datadoghq.com/DATADOG_RPM_KEY.public
      sudo zypper --non-interactive install #{node['dd-agent-step-by-step']['package_name']}
    EOF
  end
end

if node['platform_family'] != 'windows'
  execute 'add config file' do
    command <<-EOF
      sudo sh -c "sed \'s/api_key:.*/api_key: #{node['dd-agent-step-by-step']['api_key']}/\' \
      /etc/datadog-agent/datadog.yaml.example > /etc/datadog-agent/datadog.yaml"
    EOF
  end
end

if node['platform_family'] == 'windows'
end

service_provider = nil
if node['dd-agent-step-by-step']['agent_major_version'].to_i > 5 &&
  (((node['platform'] == 'amazon' || node['platform_family'] == 'amazon') && node['platform_version'].to_i != 2) ||
  (node['platform'] == 'ubuntu' && node['platform_version'].to_f < 15.04) || # chef <11.14 doesn't use the correct service provider
  (node['platform'] != 'amazon' && node['platform_family'] == 'rhel' && node['platform_version'].to_i < 7))
  # use Upstart provider explicitly for Agent 6 on Amazon Linux < 2.0 and RHEL < 7
  service_provider = Chef::Provider::Service::Upstart
end

service 'datadog-agent' do
  provider service_provider unless service_provider.nil?
  action :start
end
