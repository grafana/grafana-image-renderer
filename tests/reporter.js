import { DefaultReporter } from '@jest/reporters'

class Reporter extends DefaultReporter
{
	constructor()
	{
		super(...arguments)
	}

	printTestFileHeader(_testPath, config, result)
	{
		const console = result.console

		if(result.numFailingTests === 0 && !result.testExecError)
		{
			result.console = null
		}

		super.printTestFileHeader(...arguments)

		result.console = console
	}
}

export default Reporter
