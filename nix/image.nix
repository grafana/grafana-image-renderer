{ pkgs }:

let
  chromium = (import ./chromium.nix { inherit pkgs; }).browser;
  renderer = import ./renderer.nix { inherit pkgs; };
  fonts = [
    pkgs.dejavu_fonts
    pkgs.inter
    pkgs.liberation_ttf
    pkgs.freefont_ttf
    pkgs.noto-fonts-cjk-sans
    pkgs.noto-fonts-color-emoji
    pkgs.unifont.out
    pkgs.open-sans
    pkgs.roboto
  ];
  fontsConf = pkgs.makeFontsConf { fontDirectories = fonts; };
  locale = pkgs.glibcLocales.override {
    allLocales = false;
    locales = [ "en_US.UTF-8/UTF-8" ];
  };
in
pkgs.dockerTools.buildLayeredImage {
  name = "grafana-image-renderer";
  tag = "nix";
  contents = [
    pkgs.dockerTools.caCertificates
    pkgs.dockerTools.binSh
    pkgs.dockerTools.usrBinEnv
    (pkgs.dockerTools.fakeNss.override {
      extraPasswdLines = [ "nonroot:x:65532:65532:Renderer:/home/nonroot:/bin/sh" ];
      extraGroupLines = [ "nonroot:x:65532:" ];
    })
    (pkgs.buildEnv {
      name = "image-renderer-tools";
      paths = [
        renderer
        pkgs.coreutils
        pkgs.nssTools
        pkgs.fontconfig
        pkgs.fontconfig.out
      ];
      pathsToLink = [
        "/bin"
        "/etc/fonts"
        "/share/licenses"
      ];
      postBuild = ''
        ln -s ${chromium}/libexec/chromium/chromium $out/bin/chromium
      '';
    })
  ];
  fakeRootCommands = ''
    mkdir -p ./tmp ./home/nonroot ./run/wrappers/bin
    chmod 1777 ./tmp
    chown 65532:0 ./home/nonroot
    chmod 0775 ./home/nonroot
    cp ${chromium.sandbox}/bin/${chromium.sandboxExecutableName} ./run/wrappers/bin/
    chown 0:0 ./run/wrappers/bin/${chromium.sandboxExecutableName}
    chmod 4755 ./run/wrappers/bin/${chromium.sandboxExecutableName}
  '';
  config = {
    User = "65532";
    WorkingDir = "/home/nonroot";
    Entrypoint = [
      "${pkgs.tini}/bin/tini"
      "--"
      "/bin/grafana-image-renderer"
    ];
    Cmd = [ "server" ];
    ExposedPorts."8081/tcp" = { };
    Env = [
      "PATH=/bin"
      "HOME=/home/nonroot"
      "BROWSER_PATH=/bin/chromium"
      "CHROME_DEVEL_SANDBOX=/run/wrappers/bin/${chromium.sandboxExecutableName}"
      "LANG=en_US.UTF-8"
      "LC_ALL=en_US.UTF-8"
      "LOCALE_ARCHIVE=${locale}/lib/locale/locale-archive"
      "FONTCONFIG_FILE=${fontsConf}"
      "SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt"
      "SSL_CERT_DIR=/etc/ssl/certs"
      "TZ=UTC"
    ];
    Healthcheck = {
      Test = [
        "CMD"
        "/bin/grafana-image-renderer"
        "healthcheck"
      ];
      Interval = 10000000000;
      Timeout = 3000000000;
      StartPeriod = 30000000000;
      StartInterval = 250000000;
      Retries = 3;
    };
    Labels."org.opencontainers.image.source" = "https://github.com/grafana/grafana-image-renderer";
  };
}
