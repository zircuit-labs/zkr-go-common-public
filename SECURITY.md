# Zircuit Bug Bounty Program: Zircuit core

**Last updated on Nov 13th 2025**

## Responsible Disclosure Guidelines

- Do not disclose vulnerabilities publicly or test them on production networks. Violating this policy will forfeit your right to a reward and may put users at risk.
- Do not file public issues disclosing the vulnerability.
- Do not test vulnerabilities on public testnets or mainnets.
- Submit all findings via the approved disclosure channel below.

## Reporting Process

- All vulnerabilities must be reported via email to: bugbounty@zircuit.com
- Reports must include:
  - Impacted repository and commit SHA where the vulnerable code exists
  - A detailed description of the issue, including a justification of the severity level (impact, feasibility and likelihood).
  - Steps to reproduce
  - A working and reproducible Proof of Concept (PoC)
  - How to fix the issue
- We strongly recommend encrypting your report using our PGP public key (instructions below) to protect sensitive details
- Our team will acknowledge the reception of your submission, assess its validity, may ask for further clarification if needed and will close the topic with a final decision regarding the validity of the submission.
- Provide us with a reasonable amount of time to fix the issue before any public disclosure.

## Scope of the Program

### In-Scope Components

- Zircuit code
  - zkr-go-common-official (main branch)
  - zkr-monorepo-official (main branch)
  - l2-geth-official (main branch)
  - zr-kona-official (main branch)
  - op-succinct-official (main branch)
- Zircuit Smart Contracts
  - zkr-monorepo-official/packages/contracts-bedrock/src (main branch)
  - On-chain addresses:
    - https://docs.zircuit.com/addresses/l1-bridge
    - https://docs.zircuit.com/addresses/l2-predeploys

### Out-of-Scope

- Code that is not used in Production
- Known and previously disclosed vulnerabilities to Zircuit
- Known issues in the OP Stack, Geth, Kona, or OP-Succinct
- Best practices or non-impactful code preferences
- Experimental or undeployed features
- Front-end, infrastructure bugs and Zircuit staking (a separate policy exists: https://docs.zircuit.com/info/security/bug-bounty)
- Social engineering or physical attacks against our employees or customers
- Purely theoretical issues without a Proof of Concept
- The program covers only the code that Zircuit has modified, or added in the repositories listed above. The parts of the code that were not modified from the original upstream projects (for example: OP Stack, Geth, Kona, OP-Succinct and other “vanilla” upstream sources) are **out of scope** (we recommend reporting in a responsible manner to the responsible team) **unless** the reporter either:
  - demonstrates a vulnerability introduced by Zircuit’s modifications (i.e., the issue is present in Zircuit’s forked repository because of changes made by Zircuit), or
  - provides a working Proof-of-Concept that shows an exploit path that is only exploitable in Zircuit’s production environment (for example due to Zircuit’s configuration, bundled dependencies, packaging, integration, or feature flags).

### Unscoped findings

We welcome any relevant vulnerabilities **outside of the official scope**. Rewards will be granted **at our discretion** for significant and relevant findings. Note that a Proof of Concept is required in that case as well.

## Severity Classification System

- **Critical**:
  - Large amounts of funds permanently lost
  - Ability to cause the protocol to finalize an invalid state transition or to accept a corrupted canonical state
  - Denial of service (more than 24 hours) with broad impact (e.g., withdrawals delayed, core operations impaired) without requiring extraordinary attacker cost
- **High**:
  - Medium amounts of funds permanently lost (e.g. only affecting individual users or edge cases)
  - Large amounts of funds temporarily frozen (more than 6 hours) without requiring extraordinary attacker cost.
  - Temporary denial of service (more than 6 hours) with broad impact (e.g., withdrawals delayed, core operations impaired) without requiring extraordinary attacker cost.
- **Medium**:
  - Limited amounts of funds permanently lost (e.g., rounding errors, improper fee calculation, unlikely edge cases)
  - Medium amounts of funds temporarily frozen (more than 6 hours and only affecting individual users or edge cases) without requiring extraordinary attacker cost.

### Disclaimers

For the triage, we will consider the following aspects to assess the correctness of the claimed severity:

- **Privileges & Preconditions**: If exploit requires privileged role (owner, admin), or only works under very rare chain states, that may reduce severity.
- **Scope / Breadth**: Whether only one user / one contract is affected vs many users / core contracts / the protocol’s treasury.
- **Recovery / Mitigations**: If the protocol has pre-stated mechanisms (pausing, emergency withdrawal, ability to upgrade) that can limit damage, or if vulnerabilities are mitigated at an operational level, that may reduce severity.

## Rewards

The reward amount is determined by the **severity of the bug** and **funds at risk**. We follow an internal model to assess risk and impact.

- **Critical**: Up to $50,000
- **High**: Up to $15,000
- **Medium**: Up to $5,000

Reward amounts may vary based on factors such as:

- The exploitability of the vulnerability
- The impact on funds and system integrity
- The quality of the report and proof of concept

**For payouts, we**:

- We offer payments in USDC, USDT and USD.
- We require invoices.
- We require KYC.

## Eligibility & Legal

Participation in this program is open to security researchers, but must comply with applicable laws.

For a given issue, the first eligible and valid report submitted for the root cause of that issue will get the reward.

Any prior direct or indirect access to confidential information may lead to ineligibility to participate. If unsure, ask us by email to: bugbounty@zircuit.com.

**Participation in our Bug Bounty Program is subject to our terms and conditions. By submitting a report, you agree to these terms, which include confidentiality provisions and a release of claims against Zircuit.**

## Program Governance

Zircuit reserves the right to:

- Bypass this policy and disclose sooner if necessary for user protection
- Privately notify selected downstream users before a public disclosure
- Adjust this policy and its terms as the program evolves

## Resources

### Testing guidelines

Please do not test on production. We recommend setting up a local environment for safe testing. Local forks of Mainnet as PoCs are accepted, if they demonstrate the feasibility of the attack. For testing guidelines, please look at the README files.

### Documentation

Zircuit Developer Docs - https://docs.zircuit.com/

### PGP encryption

If you want to share an encrypted submission, we suggest the formats .txt, .md, .pdf or .zip

1. Install GnuPG
   On macOS:
   `brew install gnupg`
   On Ubuntu/Debian:
   `sudo apt-get update && sudo apt-get install -y gnupg`
   On Windows:
   `Download Gpg4win`
2. Verify the key
   Fingerprint:
   `8852ABB76E1598994AB1C5772CFF7C61FC8EDC03`
   Full key: https://keys.openpgp.org
3. Encrypt the file report.txt:
   `gpg --encrypt --armor -r bugbounty@zircuit.com report.txt`
4. Send an email to bugbounty@zircuit.com with the encrypted file attached
