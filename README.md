# GoDaddy DNS Updater for Dynamic IP

Automatically updates your GoDaddy DNS A records when your public IP address changes. Perfect for home servers or dynamic IP situations.

## Features

- Monitors your public IP address at configurable intervals
- Automatically updates GoDaddy DNS via their API when IP changes
- Logs all IP changes and updates to file
- Runs hidden in the background on Windows (no console window)
- Configurable via JSON file
- Minimal downtime - checks every few minutes

## Prerequisites

1. **Go Programming Language**: Install from [golang.org](https://golang.org/dl/)
2. **GoDaddy API Keys**: Get your API key and secret from [GoDaddy Developer Portal](https://developer.godaddy.com/keys)

## Setup Instructions

### Step 1: Get GoDaddy API Credentials

1. Go to [https://developer.godaddy.com/keys](https://developer.godaddy.com/keys)
2. Sign in with your GoDaddy account
3. Click "Create New API Key"
4. Choose **Production** environment
5. Give it a name like "DNS Updater"
6. Save your **API Key** and **API Secret** (you won't be able to see the secret again!)

### Step 2: Configure the Application

1. Copy `config.json.example` to `config.json`:
   ```
   copy config.json.example config.json
   ```

2. Edit `config.json` with your details:
   ```json
    {
    "godaddy_api_key": "YOUR_API_KEY_HERE",
    "godaddy_api_secret": "YOUR_API_SECRET_HERE",
    "domain": "YOUR_DOMAIN_NAME",
    "record_name": "@",
    "check_interval_minutes": 5,
    "ttl": 600
    }
   ```

   **Configuration Options:**
   - `godaddy_api_key`: Your GoDaddy API key
   - `godaddy_api_secret`: Your GoDaddy API secret
   - `domain`: Your domain name (e.g., "example.com")
   - `record_name`: DNS record to update ("@" for root domain, or subdomain like "www")
   - `check_interval_minutes`: How often to check for IP changes (default: 5 minutes)
   - `ttl`: DNS Time To Live in seconds (default: 600 = 10 minutes)

### Step 3: Build the Application

**Option A: Visible Console (for testing)**
```
build.bat
```
This creates `godaddy-dns-updater.exe` which shows a console window.

**Option B: Hidden Console (for production)**
```
build-hidden.bat
```
This creates `godaddy-dns-updater-hidden.exe` which runs silently in the background.

## Running the Application

### Testing (with visible console):
```
.\godaddy-dns-updater.exe
```

You'll see output like:
```
[2025-12-07 10:30:00] GoDaddy DNS Updater started
[2025-12-07 10:30:00] Monitoring domain: example.com
[2025-12-07 10:30:00] Record name: @
[2025-12-07 10:30:00] Check interval: 5 minutes
[2025-12-07 10:30:01] Current IP: 123.45.67.89
[2025-12-07 10:30:01] IP changed from  to 123.45.67.89
[2025-12-07 10:30:02] Successfully updated GoDaddy DNS
```

### Running in Background (hidden):
```
start .\godaddy-dns-updater-hidden.exe
```

The program will run silently. Check `ip_log.txt` for activity.

## Running at Startup

To run automatically when Windows starts:

1. Press `Win + R`, type `shell:startup`, press Enter
2. Create a shortcut to `godaddy-dns-updater-hidden.exe` in the startup folder
3. Right-click the shortcut → Properties → Change "Run" to "Minimized"

Alternatively, use Task Scheduler for more control:

1. Open Task Scheduler
2. Create Basic Task
3. Name: "GoDaddy DNS Updater"
4. Trigger: "When I log on"
5. Action: "Start a program"
6. Program: `C:\GoDaddy-DNS-Updater\godaddy-dns-updater-hidden.exe`
7. Start in: `C:\GoDaddy-DNS-Updater`
8. Check "Run whether user is logged on or not" for true background operation

## Log Files

The application creates two files:

- **`ip_log.txt`**: Complete log of all checks and updates with timestamps
- **`last_ip.txt`**: Stores the last known IP address

## Monitoring

Check if it's running:
```powershell
Get-Process | Where-Object {$_.ProcessName -like "*godaddy*"}
```

View recent logs:
```powershell
Get-Content ip_log.txt -Tail 20
```

## Troubleshooting

### "Error loading config"
- Make sure `config.json` exists in the same folder as the executable
- Verify the JSON syntax is correct

### "godaddy_api_key not configured"
- Replace `YOUR_API_KEY_HERE` with your actual GoDaddy API key
- Make sure there are no extra spaces

### "API request failed with status 401"
- Your API credentials are incorrect
- Generate new API keys from GoDaddy

### "API request failed with status 404"
- Check that your domain name is correct
- Verify the domain is in your GoDaddy account

### "Error getting current IP"
- Check your internet connection
- Firewall may be blocking the IP check service

## Testing the Update

To test if it works:

1. Run the visible console version
2. The first run will update the DNS (IP change from nothing to current IP)
3. Wait for the next check cycle (default 5 minutes)
4. If IP hasn't changed, you'll see "IP unchanged, no update needed"

To force an update for testing:
1. Delete `last_ip.txt`
2. Restart the program
3. It will treat the current IP as new and update DNS

## How It Works

1. On startup, loads configuration from `config.json`
2. Checks current public IP using `https://api.ipify.org`
3. Compares with last known IP from `last_ip.txt`
4. If different, updates GoDaddy DNS using their REST API
5. Logs the change and saves new IP
6. Waits for configured interval and repeats

## Security Notes

- Keep your `config.json` secure - it contains API credentials
- Don't commit `config.json` to version control
- Use the included `config.json.example` as a template
- API credentials have access to your domain settings

## Updating Subdomains

To update a subdomain instead of the root domain, change `record_name`:

```json
{
  "record_name": "www",
  ...
}
```

For multiple records, run multiple instances with different config files.

## Stopping the Program

**Visible version:** Close the console window or press `Ctrl+C`

**Hidden version:**
```powershell
Stop-Process -Name "godaddy-dns-updater-hidden"
```

Or use Task Manager to end the process.

## License

This is a personal utility. Use at your own risk. No warranty provided.

## Support

For GoDaddy API issues, see: https://developer.godaddy.com/doc/endpoint/domains

For issues with this tool, check the `ip_log.txt` file for error messages.

---

## Why I Built This

I created this tool to solve a real problem: running a web server from home without paying for expensive hosting services. With residential internet connections, your IP address changes periodically, which breaks DNS and makes your server unreachable. This tool automatically detects those changes and updates your domain's DNS records instantly, ensuring minimal downtime.

## Perfect for Self-Hosted Developers

This project is ideal for:

### **Beginners & Students**
- Learn web development without monthly hosting costs
- Run your first server, API, or website from your own computer
- Experiment freely without worrying about VPS bills or shared hosting limitations
- Great for learning DevOps, networking, and server administration concepts

### **Budget-Conscious Developers**
- **Can't afford a VPS?** Use your existing hardware instead
- A typical VPS costs £5-20/month - this solution is completely free (you just need a domain)
- Perfect for side projects, portfolios, and personal websites
- No need to pay for cloud hosting when you have a computer at home

### **Home Server Enthusiasts**
- Host your own services: personal websites, APIs, game servers, file storage, etc.
- Turn an old computer into a production server
- Full control over your infrastructure without recurring cloud costs
- Great for Raspberry Pi projects and home lab setups

### **Why This Matters**

Traditional hosting can be a barrier to entry for new developers. This tool removes that barrier by letting you:
- **Learn by doing** - Get real production experience managing your own server
- **Save money** - Especially important when you're just starting out
- **Own your infrastructure** - Complete control without relying on third parties
- **Scale gradually** - Start free at home, move to paid hosting only when you need it

Your home internet connection is powerful enough to serve websites to the world. This tool just makes sure people can always find you, no matter how often your IP changes. It's about democratizing web hosting and making development accessible to everyone, regardless of budget.

**Start building and deploying real projects today - no credit card required.**
