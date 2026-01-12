package acceptance

import (
	"math/rand/v2"
	"net/http"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegression694(t *testing.T) {
	LongTest(t)
	t.Parallel()

	t.Run("openshift: docker image is alive when using random UID", func(t *testing.T) {
		t.Parallel()
		// https://docs.redhat.com/en/documentation/openshift_container_platform/4.18/html/images/creating-images#use-uid_create-images
		//   > Because the container user is always a member of the root group, the container user can read and write these files.
		// https://www.redhat.com/en/blog/a-guide-to-openshift-and-uids
		//   > Notice the Container is using the UID from the Namespace. An important detail to notice is that the user in the Container always has GID=0, which is the root group.

		const min = 100000
		const max = 999999
		uid := rand.IntN(max-min) + min

		svc := StartImageRenderer(t, WithUser(strconv.Itoa(uid)))

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, svc.HTTPEndpoint, nil)
		require.NoError(t, err, "could not construct HTTP request to /")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "could not make HTTP request to /")
		require.Equal(t, http.StatusOK, resp.StatusCode, "expected HTTP 200 OK from /")
	})
}
