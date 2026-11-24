{ pkgs }:

pkgs.buildGo126Module {
  pname = "grafana-image-renderer";
  version = "5.0.0";

  src = pkgs.lib.cleanSource ../.;

  vendorHash = "sha256-HF/BdLJ1SMr6kgIC0VcqeVUkAyWRlgiC7wrF12NywVo=";
  env.CGO_ENABLED = 0;
  subPackages = [ "." ];
  flags = [ "-buildvcs=false" ];
  ldflags = [
    "-s"
    "-w"
  ];

  checkPhase = ''
    runHook preCheck
    go test ./... -short
    runHook postCheck
  '';

  postInstall = ''
    install -Dm644 LICENSE $out/share/licenses/grafana-image-renderer/LICENSE
  '';

  meta = {
    description = "Grafana panel and dashboard rendering service";
    homepage = "https://github.com/grafana/grafana-image-renderer";
    license = pkgs.lib.licenses.asl20;
    mainProgram = "grafana-image-renderer";
    platforms = [
      "x86_64-linux"
      "aarch64-linux"
    ];
  };
}
