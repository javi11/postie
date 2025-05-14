---
sidebar_position: 5
---

# Obfuscation Policies

Obfuscation is an important feature in Postie that helps protect your posts from automated takedown systems and provides privacy. This guide explains the different obfuscation policies available.

## Why Use Obfuscation?

Obfuscation serves several purposes:

1. **Privacy Protection**: Masks the identity of the poster
2. **Content Protection**: Makes it harder for automated systems to identify and remove content
3. **DMCA Avoidance**: Helps avoid automated DMCA takedown notices
4. **Longevity**: Increases the lifespan of your posts on Usenet

## Available Obfuscation Policies

Postie offers three levels of obfuscation that can be configured separately for regular posts and PAR2 files:

### Full Obfuscation (`full`)

This is the highest level of obfuscation:

- **Subject**: Completely obfuscated with random characters
- **Filename**: Obfuscated in yEnc headers
- **yEnc Header Filename**: Randomized for every article
- **Date**: Randomized within the last 6 hours
- **NGX Header**: Not added
- **Poster**: Random for each article

Example configuration:

```yaml
posting:
  obfuscation_policy: full
  par2_obfuscation_policy: full
```

### Partial Obfuscation (`partial`)

A moderate level of obfuscation:

- **Subject**: Obfuscated
- **Filename**: Obfuscated
- **yEnc Header Filename**: Same for all articles in a post
- **Date**: Real posted date
- **Poster**: Same for all articles in a post

Example configuration:

```yaml
posting:
  obfuscation_policy: partial
  par2_obfuscation_policy: partial
```

### No Obfuscation (`none`)

No obfuscation is applied:

- **Subject**: Original filename
- **Filename**: Original filename in yEnc headers
- **Date**: Real posted date
- **Poster**: Same for all articles, uses the value from `default_from` if provided

Example configuration:

```yaml
posting:
  obfuscation_policy: none
  par2_obfuscation_policy: none
```

## Separate Policies for Regular Files and PAR2 Files

Postie allows you to set different obfuscation policies for regular files and PAR2 recovery files:

```yaml
posting:
  obfuscation_policy: full # For regular files
  par2_obfuscation_policy: none # For PAR2 files
```

This flexibility allows you to make PAR2 files more easily discoverable while keeping your main content obfuscated.

## Best Practices

1. **Full Obfuscation for Sensitive Content**: Use full obfuscation for content that might be subject to takedown notices.
2. **Partial Obfuscation for General Use**: A good balance between protection and usability.
3. **No Obfuscation for Public Content**: Only use when posting public domain or your own original content.
4. **Mixed Strategy**: Consider using different obfuscation levels for main content vs. PAR2 files to balance discoverability with protection.
