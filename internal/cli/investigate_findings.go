package cli

import "encoding/json"

func deliveryFindings(email, stream string, evidence deliverySearchEvidence) []evidenceRecord {
	findings := []evidenceRecord{}
	if evidence.MessageTotal == 0 && evidence.BounceTotal == 0 {
		return append(findings,
			findingRecord("warning", "No outbound messages or bounces were found for this recipient in the selected window.", map[string]any{
				"email":  email,
				"stream": stream,
			}),
			nextCommandRecord("agent-postmark --server-id <server-id> streams list", "Confirm the stream used for this message."),
			nextCommandRecord("agent-postmark messages search --to <email> --stream <other-stream>", "Retry in another likely message stream."),
		)
	}
	if evidence.MessageTotal > 0 {
		findings = append(findings, findingRecord("info", "Outbound message activity was found for this recipient.", map[string]any{
			"message_total": evidence.MessageTotal,
		}))
		findings = append(findings, nextCommandRecord("agent-postmark messages get <message-id>", "Inspect the most relevant message details."))
	}
	if evidence.BounceTotal > 0 {
		severity := "warning"
		summary := "Bounce activity was found for this recipient."
		for _, bounce := range evidence.Bounces {
			if inactive, ok := bounce["Inactive"].(bool); ok && inactive {
				severity = "critical"
				summary = "Recipient appears inactive due to a bounce; future delivery may be suppressed."
				break
			}
		}
		findings = append(findings, findingRecord(severity, summary, map[string]any{
			"bounce_total": evidence.BounceTotal,
		}))
		findings = append(findings, nextCommandRecord("agent-postmark bounces get <bounce-id>", "Inspect the bounce type and inactive state."))
		findings = append(findings, nextCommandRecord("agent-postmark suppressions check <email>", "Check whether the recipient is currently suppressed."))
	}
	return findings
}

func bounceFindings(bounce map[string]any) []evidenceRecord {
	severity := "info"
	summary := "Bounce record found."
	if inactive, _ := bounce["Inactive"].(bool); inactive {
		severity = "critical"
		summary = "Recipient is inactive because of this bounce; future delivery may be suppressed."
	} else if bounce["Type"] != nil {
		severity = "warning"
		summary = "Bounce record found; inspect type and can-activate state before retrying delivery."
	}
	return []evidenceRecord{findingRecord(severity, summary, map[string]any{
		"type":         bounce["Type"],
		"name":         bounce["Name"],
		"inactive":     bounce["Inactive"],
		"can_activate": bounce["CanActivate"],
	})}
}

func domainHealthFindings(domain map[string]any, signatures []map[string]any) []evidenceRecord {
	records := []evidenceRecord{}
	checks := map[string]bool{
		"dkim":        boolValue(domain["DKIMVerified"]),
		"spf":         boolValue(domain["SPFVerified"]),
		"return_path": boolValue(domain["ReturnPathDomainVerified"]),
	}
	if checks["dkim"] && checks["spf"] && checks["return_path"] {
		records = append(records, findingRecord("ok", "Domain authentication checks are verified.", map[string]any{"checks": checks}))
	} else {
		records = append(records, findingRecord("warning", "One or more domain authentication checks are not verified.", map[string]any{"checks": checks}))
		if !checks["dkim"] {
			records = append(records, nextCommandRecord("agent-postmark domains verify-dkim <domain-id> --yes", "Ask Postmark to re-check DKIM after DNS is corrected."))
		}
		if !checks["spf"] {
			records = append(records, nextCommandRecord("agent-postmark domains verify-spf <domain-id> --yes", "Ask Postmark to re-check SPF after DNS is corrected."))
		}
	}
	if len(signatures) == 0 {
		records = append(records, findingRecord("warning", "No sender signatures were found for this domain.", nil))
	}
	return records
}

func streamHealthFindings(stream string, statsRaw, bounceRaw, webhooksRaw, suppressionRaw json.RawMessage) []evidenceRecord {
	records := []evidenceRecord{}
	stats := rawObject(statsRaw)
	inactiveMails := numberValue(stats["InactiveMails"])
	if inactiveMails > 0 {
		records = append(records, findingRecord("warning", "Stream has inactive mail count in delivery stats.", map[string]any{"stream": stream, "inactive_mails": inactiveMails}))
	}
	if bounceTotal := rawTotal(bounceRaw); bounceTotal > 0 {
		records = append(records, findingRecord("warning", "Recent bounces were found for this stream.", map[string]any{"stream": stream, "bounce_total": bounceTotal}))
		records = append(records, nextCommandRecord("agent-postmark bounces list --stream "+stream, "Inspect recent bounces by type and inactive state."))
	}
	if suppressionTotal := rawTotal(suppressionRaw); suppressionTotal > 0 {
		records = append(records, findingRecord("info", "Suppressions exist for this stream.", map[string]any{"stream": stream, "suppression_total": suppressionTotal}))
	}
	coverage := webhookCoverage(rawEnvelopeList(webhooksRaw, "Webhooks"))
	records = append(records, webhookCoverageFinding(coverage))
	if missingWebhookCoverage(coverage) {
		records = append(records, nextCommandRecord("agent-postmark webhooks health", "Inspect webhook trigger coverage."))
	}
	if len(records) == 0 {
		records = append(records, findingRecord("ok", "No obvious stream health issues found in delivery stats, bounces, suppressions, or webhooks.", map[string]any{"stream": stream}))
	}
	return records
}

func writeWebhookEvidence(raw json.RawMessage) error {
	rows := rawEnvelopeList(raw, "Webhooks")
	records := []evidenceRecord{entityRecord("webhooks", nil, compactRows("Webhooks", rows))}
	records = append(records, webhookCoverageFinding(webhookCoverage(rows)))
	return writeEvidence(records)
}

func webhookCoverageFinding(coverage map[string]int) evidenceRecord {
	if missingWebhookCoverage(coverage) {
		return findingRecord("warning", "Webhook coverage is missing delivery or bounce triggers.", map[string]any{"coverage": coverage})
	}
	return findingRecord("ok", "Webhook coverage includes delivery and bounce triggers.", map[string]any{"coverage": coverage})
}

func missingWebhookCoverage(coverage map[string]int) bool {
	return coverage["delivery"] == 0 || coverage["bounce"] == 0
}

func webhookCoverage(rows []json.RawMessage) map[string]int {
	coverage := map[string]int{"delivery": 0, "bounce": 0, "inbound": 0, "spam_complaint": 0}
	for _, row := range rows {
		obj := rawObject(row)
		triggers, _ := obj["Triggers"].(map[string]any)
		for key, value := range triggers {
			if !boolValue(value) {
				continue
			}
			switch key {
			case "Delivery":
				coverage["delivery"]++
			case "Bounce":
				coverage["bounce"]++
			case "Inbound":
				coverage["inbound"]++
			case "SpamComplaint":
				coverage["spam_complaint"]++
			}
		}
	}
	return coverage
}
