# Raspberry Pi Ansible Configuration

This directory contains Ansible configuration for managing Raspberry Pi devices.

## Setup Instructions

1. Copy `inventory/hosts.template` to `inventory/hosts`
2. Update the `hosts` file with your Pi information:
   - IP addresses
   - Username
   - SSH key path

## Security Notes

- The `hosts` file is gitignored to prevent committing sensitive information
- Never commit files containing:
  - IP addresses
  - Usernames
  - Private keys
  - Passwords
  - Other credentials

## Directory Structure

```
ansible/
├── inventory/
│   ├── hosts (gitignored)
│   └── hosts.template
├── playbooks/
│   └── test_connection.yml
└── roles/
```

## Usage

Test connection to all Pis:
```bash
ansible-playbook -i inventory/hosts playbooks/test_connection.yml
```

## Adding New Devices

1. Add the device's SSH key:
   ```bash
   ssh-copy-id -i ~/.ssh/id_ed25519.pub username@device_ip
   ```
2. Add the device to your local `inventory/hosts` file
3. Test the connection using the test playbook 