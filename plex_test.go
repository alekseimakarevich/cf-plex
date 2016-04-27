package main_test

import (
	. "github.com/EngineerBetter/cf-plex"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gbytes"
	. "github.com/onsi/gomega/gexec"

	"io"
	"io/ioutil"
	"os"
	"time"
)

var _ = Describe("cf-plex", func() {
	var tmpDir string
	var cfUsername string
	var cfPassword string

	BeforeEach(func() {
		var err error
		tmpDir, err = ioutil.TempDir("", "plex")
		Ω(err).ShouldNot(HaveOccurred())

		cfUsername = os.Getenv("CF_USERNAME")
		if cfUsername == "" {
			cfUsername = "testing@engineerbetter.com"
		}

		cfPassword = os.Getenv("CF_PASSWORD")
		Ω(cfPassword).ShouldNot(BeZero(), "CF_PASSWORD env var must be set")
	})

	AfterEach(func() {
		Ω(os.RemoveAll(tmpDir)).Should(Succeed())
	})

	Describe("SetEnv", func() {
		Context("when the env var is already set", func() {
			It("replaces the value", func() {
				env := []string{"KEY=value", "CF_HOME=foo"}
				env = SetEnv("CF_HOME", "bar", env)
				Ω(env).Should(ContainElement("CF_HOME=bar"))
				Ω(env).Should(ContainElement("KEY=value"))
				Ω(env).ShouldNot(ContainElement("CF_HOME=foo"))
			})
		})

		Context("when the env var is not set already", func() {
			It("adds the value", func() {
				env := []string{"KEY=value"}
				env = SetEnv("CF_HOME", "bar", env)
				Ω(env).Should(ContainElement("CF_HOME=bar"))
				Ω(env).Should(ContainElement("KEY=value"))
			})
		})
	})

	It("calls external things", func() {
		env := os.Environ()
		env = SetEnv("CF_PLEX_HOME", tmpDir, env)
		cliPath, err := Build("github.com/EngineerBetter/cf-plex")
		Ω(err).ShouldNot(HaveOccurred())

		session, err := Start(CommandWithEnv(env, cliPath, "apps"), GinkgoWriter, GinkgoWriter)
		Ω(err).ShouldNot(HaveOccurred())
		session.Wait("1s")
		Ω(session.Err).Should(Say("No APIs have been set"))

		addApi("https://api.run.pivotal.io", cfUsername, cfPassword, env, cliPath)
		addApi("https://api.eu-gb.bluemix.net", cfUsername, cfPassword, env, cliPath)

		session, err = Start(CommandWithEnv(env, cliPath, "list-apis"), GinkgoWriter, GinkgoWriter)
		Ω(err).ShouldNot(HaveOccurred())
		session.Wait("1s")
		Ω(session.Out).Should(Say("https___api.run.pivotal.io"))

		cmd := CommandWithEnv(env, cliPath, "delete-org", "does-not-exist")
		in, _ := cmd.StdinPipe()
		Ω(err).ShouldNot(HaveOccurred())
		session, err = Start(cmd, GinkgoWriter, GinkgoWriter)
		Ω(err).ShouldNot(HaveOccurred())

		confirm("Really delete the org does-not-exist and everything associated with it?", "n", session, in)
		Eventually(session, "5s").Should(Say("Delete cancelled"))

		confirm("Really delete the org does-not-exist and everything associated with it?", "n", session, in)
		Eventually(session, "5s").Should(Say("Delete cancelled"))
		Eventually(session).Should(Exit(0))

		removeApi("https://api.run.pivotal.io", env, cliPath)
		removeApi("https://api.eu-gb.bluemix.net", env, cliPath)

		session, err = Start(CommandWithEnv(env, cliPath, "apps"), GinkgoWriter, GinkgoWriter)
		Ω(err).ShouldNot(HaveOccurred())
		session.Wait("1s")
		Ω(session.Err).Should(Say("No APIs have been set"))
	})

	It("fails when subprocesses fail", func() {
		env := os.Environ()
		env = SetEnv("CF_PLEX_HOME", tmpDir, env)
		cliPath, err := Build("github.com/EngineerBetter/cf-plex")
		Ω(err).ShouldNot(HaveOccurred())
		session, err := Start(CommandWithEnv(env, cliPath, "rubbish"), GinkgoWriter, GinkgoWriter)
		Eventually(session).Should(Exit(1))
	})
})

func addApi(api, cfUsername, cfPassword string, env []string, cliPath string) {
	session, err := Start(CommandWithEnv(env, cliPath, "add-api", api, cfUsername, cfPassword), GinkgoWriter, GinkgoWriter)
	Ω(err).ShouldNot(HaveOccurred())
	session.Wait("5s")
	Ω(session.Out).Should(Say("Setting api endpoint to " + api + "...\nOK"))
	Ω(session.Out).Should(Say("Authenticating...\nOK"))
}

func removeApi(api string, env []string, cliPath string) {
	session, err := Start(CommandWithEnv(env, cliPath, "remove-api", api), GinkgoWriter, GinkgoWriter)
	Ω(err).ShouldNot(HaveOccurred())
	session.Wait("1s")
	Ω(session.Out).Should(Say("Removed " + api))
}

func confirm(expectedPrompt, input string, session *Session, in io.Writer) {
	time.Sleep(1 * time.Second)
	Ω(session).Should(Say(expectedPrompt))
	in.Write([]byte(input + "\n"))
}
