# Rules and Meta API access log

## Official Rules — read 28 October 2026

Source: "Hackathon Terms & Conditions ("Official Rules") and Event Waiver" at
<https://cerebralvalley.ai/e/meta-ai-developer-hackathon/waiver> (JavaScript page,
read by rendering it in headless Chrome; the page is dated 25 October 2026).
Themes come from <https://dev.meta.ai/events/global-hackathon>.

**Judging (projects):** Impact 30% (real-world potential, who benefits, fits a
theme), Use of Meta AI Tooling 25% (creative use, beyond a basic integration,
surfaces unexpected capabilities), Technical Depth & Execution 20% (sound,
refined engineering), Demo 25% (working, impressive demo that holds up live,
and the quality of the demo video).

**Themes:** "Build for Your World" (bring Meta's models into your domain, solve
a problem you know firsthand) or "Push the Frontier". FrameCheck fits the first.

**Submission:** through the Cerebral Valley portal
<https://cerebralvalley.ai/e/meta-ai-developer-hackathon/hackathon/submit> (login
required; its form fields were not visible to us). Due 4 November 2026, 21:00 ET.
Indie track: individuals or teams of up to four; one track per submission; the
representative names all team members.

**Tools:** "Participants build with Meta AI products, including Muse Spark,
Glimmer, SAM 3, and any other Meta AI products identified on the Event page."
Credits: US$150 of Meta Model API credits, "valid for a 10-day development
window", subject to separate Credit Terms (a PDF linked from the rules page).

**Ownership and originality:** the entrant keeps IP but grants the Organizers a
perpetual, royalty-free licence to use and publish the submission. The
submission must be the entrant's original work, solely owned, and must not
violate others' IP, contract or privacy rights. Third-party technical
assistance is allowed if the components remain the entrant's own work. Open
source may be included if its licences are followed.
Materials shared in the event are **not confidential**.

**Not stated in the rules:** any rule on code written before kickoff, a
required demo-video length, a required public repository, or a required hosted
deployment. Check the submission form for these fields.

**Eligibility:** 18+ and of age of majority; not in a sanctioned country;
not an employee, contractor or intern of Meta or Cerebral Valley; not
government-affiliated (including immediate family and household).

### Consequences for FrameCheck

- Do not put screenshots of the user's workplace software in the public
  submission, repo or video: it is not the entrant's IP and contains personal
  data. Demo with the self-owned "Lembar" pages.
- Third-party dependencies are open source (MIT/BSD/Apache; fonts under OFL);
  list them with their licences in the repository.
- Strengthen "Use of Meta AI Tooling": today one Muse Spark call per run.
- "Holds up live" and video quality weigh 25%: a hosted demo and a polished
  video matter.

## Earlier notes (22 October 2026)

Checked **22 October 2026** against the public [Meta event page](https://dev.meta.ai/events/global-hackathon)
and [Cerebral Valley event page](https://cerebralvalley.ai/e/meta-ai-developer-hackathon).
The Meta page still says Official Rules will be published before the 26 October
kickoff and accepted by selected participants. No public rule text was found on
either page during this check. Therefore prior-code eligibility, mandatory model
combination, final judging rubric, and submission artifact requirements remain
**unverified**. Event marketing text is not a mandatory model rule.

The public pages list kickoff on 26 October and submissions due 4 November at
21:00 ET. The personal target is 4 November 21:00 WIB, twelve hours ahead of
the corresponding 5 November 09:00 WIB deadline. The Cerebral Valley page labels
its event time EST; the precise kickoff hour needs confirmation in the
participant agenda or calendar because October precedes the US daylight-saving
change. Use the official agenda when available.

## Access status

- Participant approval: reported from the user's Cerebral Valley email.
- Meta Model API dashboard: user reports it opens.
- Last billing screenshot: “No payment method on file.” Credit balance,
  activation, expiry, card requirement, and post-credit behavior are unverified.
- [Meta for Developers credit announcement](https://www.reddit.com/user/MetaforDevelopers/comments/1x51k0q/what_can_you_build_with_our_newest_models_and_ten/)
  says US$150 per selected participant; it is not evidence that this account
  can make a covered request now.
- No API key was read, printed, stored, or used for a request in this preparation.

## Verified public API documentation, not account access

- [Meta Model API overview](https://dev.meta.ai/docs/overview) lists base URL
  `https://api.meta.ai/v1`, Muse Spark `muse-spark-1.3`, and SAM `sam-3.1`.
- [Image understanding](https://dev.meta.ai/docs/image-understanding) shows
  image input for `muse-spark-1.3` on `/v1/responses` and explains normalized
  `0..1000` coordinate grounding.
- [Media segmentation](https://dev.meta.ai/docs/media-segmentation) shows
  `sam-3.1` on `/v1/responses` returning boxes/masks for prompted objects.
  It does not establish reliable detection of arbitrary UI components.
- [Structured output](https://dev.meta.ai/docs/structured-output) describes
  schema-constrained responses. Semantic correctness and evidence still need
  backend validation.

## Next rules check

On or after 26 October, inspect the participant portal/email and public page.
Record the rule document URL/version, accepted date, build-start definition,
reuse policy, required models, submission format, and allowed deployment. Until
then, this repository's 22 October work is preparation documentation and local
configuration, logged in `PROGRESS.md` for declaration if required.

## 27 October 2026 — participant credit claim screenshots

The user supplied screenshots of a Cerebral Valley event email, the Meta Model
API dashboard URL, and a Google form titled “Meta Global AI Hackathon: Model API
Credit Claims.” The Team ID entered in the form visually matches the `team_id`
parameter in the Meta dashboard URL. The form instructs accepted participants
to use the same email address as the hackathon application; the screenshots do
not show the original application email for independent comparison. The visible
“Kirim” button means submission is not proven by these screenshots.

The email and form say the US$150 credits are applied in batches, within two
business days after a submitted claim according to the form, and must be
checked on the Billing page. The form says a payment method is required on file
to make the first API call; the email specifically says a credit card must be
applied. The claim deadline shown is 4 November 2026 at 21:00 ET, equivalent
to 5 November 2026 at 09:00 WIB. The screenshots do not prove credits have
landed, a card has been added, or any API request has succeeded.

The email suggests a “Spend Limit” notification. [Meta's billing help](https://dev.meta.ai/help/billing/spend-limit-alert)
states this is an alert, **not a hard spending cap**; usage above the alert can
still be billed. Do not rely on it to prevent charges. The user's standing
instruction remains: no payment method, purchase, or paid API use without
authorization. Rules acceptance in the form concerns credit terms and is not
evidence that the hackathon Official Rules have been inspected or accepted.

The user then confirmed submission and supplied a screenshot of the Google
Form `formResponse` page on 27 October. Treat the **credit claim as submitted**.
The response page says granted credits will appear under Billing → free credits
and warns that API calls can charge the card before credits are applied. No
Billing balance or grant confirmation is shown, so **credit activation remains
unverified**. No API call is authorized on the basis of this form response.
