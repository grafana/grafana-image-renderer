package acceptance

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChromiumInstalled(t *testing.T) {
	LongTest(t)
	t.Parallel()

	exitCode, logs := RunImageRendererWithCommand(t, []string{"chromium", "--version"}, nil)
	require.Zero(t, exitCode, "chromium did not exist in the container (--version failed)")
	require.Regexp(t, `^Chromium [0-9]+`, logs, "chromium --version did not return expected output")
}

func TestChromiumWebGL(t *testing.T) {
	LongTest(t)
	t.Parallel()

	// Check actual rendering, including software rendering on hosts without a GPU.
	const page = `data:text/html,<html><body><canvas id="canvas" width="1" height="1"></canvas><script>
const gl = document.getElementById("canvas").getContext("webgl");
if (gl) {
  gl.clearColor(1, 0, 0, 1);
  gl.clear(gl.COLOR_BUFFER_BIT);
  const pixel = new Uint8Array(4);
  gl.readPixels(0, 0, 1, 1, gl.RGBA, gl.UNSIGNED_BYTE, pixel);
  document.body.textContent = pixel.join(",");
} else {
  document.body.textContent = "WebGL unavailable";
}
</script></body></html>`
	exitCode, logs := RunImageRendererWithCommand(t, []string{"chromium"}, []string{
		"--headless", "--no-sandbox", "--disable-dev-shm-usage", "--dump-dom", page,
	})
	require.Zero(t, exitCode, "Chromium WebGL rendering failed: %s", logs)
	require.Contains(t, logs, "<body>255,0,0,255</body>", "WebGL did not render a red pixel")
}
