class LogsMcpServer < Formula
  desc "IBM Cloud Logs MCP Server - Model Context Protocol server for IBM Cloud Logs"
  homepage "https://github.com/tareqmamari/logs-mcp-server"
  version "0.1.0"
  license "UNLICENSED"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/tareqmamari/logs-mcp-server/releases/download/v#{version}/logs-mcp-server_#{version}_Darwin_arm64.tar.gz"
      sha256 "" # Will be updated automatically by GoReleaser
    else
      url "https://github.com/tareqmamari/logs-mcp-server/releases/download/v#{version}/logs-mcp-server_#{version}_Darwin_x86_64.tar.gz"
      sha256 "" # Will be updated automatically by GoReleaser
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/tareqmamari/logs-mcp-server/releases/download/v#{version}/logs-mcp-server_#{version}_Linux_arm64.tar.gz"
      sha256 "" # Will be updated automatically by GoReleaser
    else
      url "https://github.com/tareqmamari/logs-mcp-server/releases/download/v#{version}/logs-mcp-server_#{version}_Linux_x86_64.tar.gz"
      sha256 "" # Will be updated automatically by GoReleaser
    end
  end

  def install
    bin.install "logs-mcp-server"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/logs-mcp-server --version 2>&1", 1)
  end

  def caveats
    <<~EOS
      To use logs-mcp-server, you need to configure:
        1. Get your IBM Cloud API key: https://cloud.ibm.com/iam/apikeys
        2. Set environment variables:
           export LOGS_API_KEY='your-api-key'
           export LOGS_SERVICE_URL='https://[instance-id].api.[region].logs.cloud.ibm.com'
           export LOGS_REGION='us-south'
        3. Configure in Claude Desktop

      For more information, see: https://github.com/tareqmamari/logs-mcp-server
    EOS
  end
end
