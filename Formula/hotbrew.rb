class Hotbrew < Formula
  desc "Your morning, piping hot. A beautiful terminal newsletter."
  homepage "https://github.com/jcornudella/hotbrew"
  license "MIT"
  head "https://github.com/jcornudella/hotbrew.git", branch: "main"

  depends_on "go" => :build

  def install
    ldflags = %W[
      -s -w
      -X main.Version=#{version}
    ]
    system "go", "build", *std_go_args(ldflags:), "./cmd/hotbrew"
  end

  def caveats
    <<~EOS
      ☕ hotbrew installed!

      Run 'hotbrew' to start your morning digest.

      To show hotbrew every time you open a terminal, add to your ~/.zshrc:
        command -v hotbrew &>/dev/null && hotbrew

      Config file: ~/.config/hotbrew/hotbrew.yaml
    EOS
  end

  test do
    assert_match "hotbrew v", shell_output("#{bin}/hotbrew version")
  end
end
