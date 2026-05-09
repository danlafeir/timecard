# timecard

*I built this because I was frustrated with filling out timecards at work and wanted to make it easier*

This is a CLI for submitting a timecard for the week. It currently supports an integration with the [Tempo API](https://apidocs.tempo.io/). I store secrets in your MacOS keychain and configuration at `$HOME/.timecard/`.

I am trying to create a constellation of CLI tools that make my life easier. 

## Installation

You can install the `timecard` binary with:

```sh
curl -sSL https://raw.githubusercontent.com/danlafeir/timecard/main/scripts/install.sh | sh
```

This script will detect your OS and architecture, download the binary (`timecard-<os>-<arch>`) and install it as `timecard` to `~/.local/bin`. Ensure `~/.local/bin` is in your PATH.

## Usage

**Important:** Before using this tool, you need to have at least one timecard entry filled out in Tempo within the past two weeks. This is required because the tool automatically fetches your most recent issue ID from your worklog entries to use as the default issue ID.

### Configuration

#### Getting Your Tempo API Token
To create an API token in Tempo:

1. Log in to your Atlassian account and navigate to Tempo
2. Go to **Settings** → **API Integration** (or **Profile** → **API tokens**)
3. Click **Create API Token** or **New Token**
4. Give your token a descriptive name (e.g., "timecard-cli")
5. Ensure the token has permission to manage **Worklogs**
6. Copy the generated token (you won't be able to see it again)

**Important:** The API token must have permissions to manage Worklogs for the tool to function correctly.

#### Getting Your Account ID from Atlassian
To find your Account ID (also known as Atlassian Account ID):

1. Log in to your Atlassian instance and select the JIRA app
2. Click on your profile picture/avatar in the top-right corner
3. Select **Profile**
4. Your Account ID will be displayed in URL path after `/people/<account_id>?...`

**Note:** This Account ID is associated with your profile in JIRA/Atlassian and is different from your username or email.

#### Running Configuration
To configure your Tempo API token and account ID.


**If installed as `timecard` binary:**
```sh
timecard configure --token <YOUR_TOKEN> --account-id <ACCOUNT_ID>
```

You can also omit flags to be prompted interactively.

- The API token is stored securely in the device's secure storage.
- The account ID and default issue ID are stored in the config file under a `tempo` key at `$HOME/.timecard/config.yaml`.

The issue ID will be automatically fetched from your most recent Tempo worklog entry (within the past two weeks). Make sure you assigned to the JIRA Project and use a JIRA card that belongs to the appropiate project.

### Available Commands

#### `add-week`
Add a time entry for the current week (or a past week) to Tempo. This is the main command for logging time.

The command will:
1. Prompt you to confirm the week (defaults to current week, or you can specify weeks back)
2. Ask for time spent in three categories:
   - Development/design/testing (capitalizable time)
   - PTO (vacation or sick time)
   - Other time
3. Submit all time entries to Tempo via the API

#### `configure`
Set up your API token and Account Id

Options:
- `--token` - Tempo API token
- `--account-id` - Your Tempo account ID (from JIRA)

### Hidden Commands
- `get-week` — Fetch your current week's timecard from the Tempo API (for debugging)
> I am still working on this 

## Development

### Prerequisites
- Go 1.24.3+
- MacOS because I have not generalized the secrets interaction


### Building

Build for your current OS and architecture:
```sh
make build
```

Build for all supported OS/arch:
```sh
make build-all
```

### Running Locally
After building, run the CLI:
```sh
./bin/timecard
```

### Testing
Run all tests:
```sh
make test
```

## Notes
- Only MacOS is currently supported for secure secrets storage.
- For other OS support, contributions are welcome!

---

For more information, see the code and comments or open an issue.
