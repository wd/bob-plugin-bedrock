class BobBedrockBridge < Formula
  desc "Bridge server connecting Bob Translator to AWS Bedrock Claude models"
  homepage "https://github.com/wd/bob-plugin-bedrock"
  license "MIT"
  head "https://github.com/wd/bob-plugin-bedrock.git", branch: "main"

  depends_on "go" => :build

  def install
    cd "server" do
      system "go", "build", *std_go_args(ldflags: "-s -w"), "-o", bin/"bob-bedrock-bridge", "."
    end

    # Install sample config
    (share/"bob-bedrock-bridge").mkpath
    (share/"bob-bedrock-bridge/config.sample").write <<~EOS
      # Bob Bedrock Bridge Configuration
      # Location: ~/.config/bob-bedrock-bridge/config

      # AWS Profile (matches ~/.aws/config)
      # Leave empty for default credential chain
      AWS_PROFILE=

      # AWS Region for Bedrock
      AWS_REGION=us-east-1

      # Server port
      SERVER_PORT=18081
    EOS

    # Install wrapper script
    (bin/"bob-bedrock-bridge-wrapper").write <<~BASH
      #!/bin/bash
      CONFIG_DIR="${HOME}/.config/bob-bedrock-bridge"
      CONFIG_FILE="${CONFIG_DIR}/config"

      # Create config dir
      [[ -d "$CONFIG_DIR" ]] || mkdir -p "$CONFIG_DIR"

      # Copy sample config if needed
      if [[ ! -f "$CONFIG_FILE" ]]; then
        cp "#{share}/bob-bedrock-bridge/config.sample" "$CONFIG_FILE" 2>/dev/null || true
        echo "[bob-bedrock-bridge] Created config at $CONFIG_FILE" >&2
      fi

      # Load config (ignore errors)
      if [[ -f "$CONFIG_FILE" ]]; then
        set -a
        source "$CONFIG_FILE" 2>/dev/null || true
        set +a
      fi

      exec "#{opt_bin}/bob-bedrock-bridge" "$@"
    BASH
  end

  def post_install
    (var/"log/bob-bedrock-bridge").mkpath
  end

  def caveats
    <<~EOS
      Configuration: ~/.config/bob-bedrock-bridge/config

      Edit config to set AWS_PROFILE, then restart service:
        brew services restart bob-bedrock-bridge

      For SSO authentication:
        aws sso login --profile your-profile-name

      Logs: #{var}/log/bob-bedrock-bridge/
      Endpoint: http://localhost:18081
    EOS
  end

  service do
    run [opt_bin/"bob-bedrock-bridge-wrapper"]
    keep_alive crashed: true
    working_dir "~"
    log_path var/"log/bob-bedrock-bridge/output.log"
    error_log_path var/"log/bob-bedrock-bridge/error.log"
  end

  test do
    assert_predicate bin/"bob-bedrock-bridge", :executable?
    assert_path_exists share/"bob-bedrock-bridge/config.sample"
  end
end
