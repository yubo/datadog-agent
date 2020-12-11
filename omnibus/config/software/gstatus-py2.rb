
name "gstatus-py2"
dependency "pip2"

default_version "1.0.4"
relative_path "gstatus-#{version}"

source :url => "https://github.com/gluster/gstatus/archive/v#{version}.tar.gz",
       :sha256 => "774e219c0cba8c38d2731938cd10db76d3371d744c2e4cde54c32de95a179184",
       :extract => :seven_zip

build do
  ship_license "https://raw.githubusercontent.com/gluster/gstatus/v#{version}/LICENSE"
  command "make gen-version", :env => {"VERSION" => "#{version}"}
  command "#{install_dir}/embedded/bin/pip2 install ."
end
