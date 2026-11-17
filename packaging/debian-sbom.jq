# Preserve Debian package identities for the files copied into the Wolfi image.
# In particular, Debian's security feed tracks Chromium's Debian revision.
[inputs | split("\t") | {
  SPDXID: ("SPDXRef-" + .[0]),
  name: .[0],
  versionInfo: .[1],
  downloadLocation: "NOASSERTION",
  filesAnalyzed: false,
  licenseConcluded: "NOASSERTION",
  licenseDeclared: "NOASSERTION",
  copyrightText: "NOASSERTION",
  externalRefs: [{
    referenceCategory: "PACKAGE-MANAGER",
    referenceType: "purl",
    referenceLocator: ("pkg:deb/debian/" + .[0] + "@" + (.[1] | @uri) +
      "?arch=" + .[2] + "&distro=debian-13&upstream=" + ((.[3] + "@" + .[4]) | @uri))
  }]
}] as $packages |
{
  spdxVersion: "SPDX-2.3",
  dataLicense: "CC0-1.0",
  SPDXID: "SPDXRef-DOCUMENT",
  name: "Grafana Image Renderer Debian runtime components",
  documentNamespace: ("https://grafana.com/sbom/image-renderer/debian/" +
    ($packages[0].versionInfo | @uri) + "/" + $arch + "/" + $created),
  creationInfo: {creators: ["Tool: dpkg-query"], created: $created},
  packages: $packages,
  relationships: [$packages[] | {
    spdxElementId: "SPDXRef-DOCUMENT",
    relationshipType: "DESCRIBES",
    relatedSpdxElement: .SPDXID
  }]
}
