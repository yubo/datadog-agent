#
# Cookbook Name:: dd-agent-install
# Recipe:: _repair_windows_install
#
# Copyright (C) 2021-present Datadog

# msiexec options following recommendations from https://wixtoolset.org/documentation/manual/v3/customactions/wixfailwhendeferred.html
# /f [p|o|e|d|c|a|u|m|s|v] repairs a product
# v - Runs from source and re-caches the local package.
# e - Reinstalls if file is missing or an equal or older version is installed.
# c - Reinstalls if file is missing or the stored checksum does not match the calculated value.
# m - Rewrites all required computer-specific registry entries.
# u - Rewrites all required user-specific registry entries.
# s - Overwrites all existing shortcuts.
powershell_script "repair-agent" do
  code <<-EOF
  $product_code = (Get-WmiObject Win32_Product | Where-Object -Property Name -eq 'Datadog Agent').IdentifyingNumber
  Start-Process msiexec.exe -Wait -ArgumentList '/q','/log','repair.log','/fvecmus',$product_code
  EOF
end
