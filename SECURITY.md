# Security Policy

## Zircuit Bug Bounty Program
_Warning: Do not disclose vulnerabilities publicly or by executing them against a production network. If you do, you will not only be putting users at risk, but you will forfeit your right to a reward. Always follow the appropriate reporting pathways as described below._
- _Do not disclose the vulnerability publicly, for example by filing a public issue._
- _Do not test the vulnerability on a publicly available network, either the testnet or the mainnet._

The Zircuit Bounty Program offers a reward for critical vulnerabilities found in the Zircuit codebase. The bug bounty amount will be determined based on the severity of the bug, and the amount of funds at risk. Vulnerabilities for the bug bounty program can be reported to bugbounty@zircuit.com.

**The scope includes:**
- Zircuit node and shared common code
    - https://github.com/zircuit-labs/zkr-monorepo-public
    - https://github.com/zircuit-labs/l2-geth-public
    - https://github.com/zircuit-labs/zkr-monorepo-public 
- Zircuit smart contracts
    - https://github.com/zircuit-labs/zkr-monorepo-public/tree/develop/packages/contracts-bedrock 

**The scope of the bug bounty program explicitly excludes:**
- Known and previously disclosed vulnerabilities to Zircuit
- Known vulnerabilities in the OP stack
- Known vulnerabilities in the Geth repository
- Best practices, coding preferences, and other issues without a practical impact
- Experimental features and features in development that are not deployed to mainnet
- Vulnerabilities that were exploited or otherwise violated the principles of responsible disclosure

The front-end and front-end infrastructure code bug bounty program is directed by a separate policy described on [this page](https://app.zircuit.com/faq).

### Unscoped Bugs
If you believe that you have found a significant bug or vulnerability in Zircuits smart contracts, node, infrastructure, etc., even if that component is not covered by the existing bug bounty scope, please report it to via bugbounty@zircuit.com. The Zircuit team will assess the impact of such vulnerabilities and will make decisions on the rewards on a case-by-case basis.

### Rights of maintainers
Alongside this policy, maintainers also reserve the right to:
- Bypass this policy and publish details on a shorter timeline.
- Directly notify a subset of downstream users prior to making a public announcement.