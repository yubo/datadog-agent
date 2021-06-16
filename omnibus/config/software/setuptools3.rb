name "setuptools3"
default_version "57.0.0"
skip_transitive_dependency_licensing true

dependency "python3"

relative_path "setuptools-#{version}"

source :url => "https://github.com/pypa/setuptools/archive/v#{version}.tar.gz",
       :sha256 => "283e97578ebe125dee7b64bc5b12d7cb1252a61284e7a50c6bbad294361041a9",
       :extract => :seven_zip

build do
  if ohai["platform"] == "windows"
    python_bin = "#{windows_safe_path(python_3_embedded)}\\python.exe"
    python_prefix = "#{windows_safe_path(python_3_embedded)}"
  else
    python_bin = "#{install_dir}/embedded/bin/python3"
    python_prefix = "#{install_dir}/embedded"
  end

  ship_license "PSFL"
  command "#{python_bin} -mensurepip"
  command "#{python_bin} -mpip install --prefix=#{python_prefix} ."

  if ohai["platform"] != "windows"
    block do
      FileUtils.rm_f(Dir.glob("#{install_dir}/embedded/lib/python3.*/site-packages/setuptools/*.exe"))
    end
  end
end
