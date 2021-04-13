class Chef
    class DDSystemProbeCheck
        class << self
            def arm?(node)
                %w{armv6l armv7l armhf aarch64 arm64 arch64}.include?(node["kernel"]["machine"])
            end
        end
    end
end
