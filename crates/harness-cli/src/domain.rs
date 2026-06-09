use std::fmt;
use std::str::FromStr;

use thiserror::Error;

#[derive(Debug, Error, PartialEq, Eq)]
pub enum ParseHarnessValueError {
    #[error("unknown intake type '{0}'. Use: new spec, spec slice, change request, new initiative, maintenance request, or harness improvement")]
    InputType(String),
    #[error("unknown lane '{0}'. Use: tiny, normal, or high-risk. Use tiny instead of low.")]
    RiskLane(String),
    #[error("{0} must be an integer")]
    Integer(String),
    #[error("unknown guardrail status '{0}'. Use: active or superseded")]
    GuardrailStatus(String),
    #[error("{0} must be 0 or 1. Example: --unit 1 --integration 1 --e2e 0 --platform 0")]
    BoolFlag(String),
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum InputType {
    NewSpec,
    SpecSlice,
    ChangeRequest,
    NewInitiative,
    Maintenance,
    HarnessImprovement,
}

impl InputType {
    pub fn as_db_value(&self) -> &'static str {
        match self {
            Self::NewSpec => "new_spec",
            Self::SpecSlice => "spec_slice",
            Self::ChangeRequest => "change_request",
            Self::NewInitiative => "new_initiative",
            Self::Maintenance => "maintenance",
            Self::HarnessImprovement => "harness_improvement",
        }
    }
}

impl FromStr for InputType {
    type Err = ParseHarnessValueError;

    fn from_str(value: &str) -> Result<Self, Self::Err> {
        let normalized = normalize_token(value);
        match normalized.as_str() {
            "new_spec" => Ok(Self::NewSpec),
            "spec_slice" => Ok(Self::SpecSlice),
            "change_request" => Ok(Self::ChangeRequest),
            "new_initiative" => Ok(Self::NewInitiative),
            "maintenance" | "maintenance_request" => Ok(Self::Maintenance),
            "harness_improvement" => Ok(Self::HarnessImprovement),
            _ => Err(ParseHarnessValueError::InputType(value.to_owned())),
        }
    }
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum RiskLane {
    Tiny,
    Normal,
    HighRisk,
}

impl RiskLane {
    pub fn as_db_value(&self) -> &'static str {
        match self {
            Self::Tiny => "tiny",
            Self::Normal => "normal",
            Self::HighRisk => "high_risk",
        }
    }
}

impl FromStr for RiskLane {
    type Err = ParseHarnessValueError;

    fn from_str(value: &str) -> Result<Self, Self::Err> {
        let normalized = normalize_token(value);
        match normalized.as_str() {
            "tiny" => Ok(Self::Tiny),
            "normal" => Ok(Self::Normal),
            "high_risk" => Ok(Self::HighRisk),
            _ => Err(ParseHarnessValueError::RiskLane(value.to_owned())),
        }
    }
}

pub const RISK_LANE_HELP: &str =
    "Accepted lanes: tiny, normal, high-risk. Use tiny instead of low.";

#[derive(Debug, PartialEq, Eq)]
pub struct IntakeRecord {
    pub id: i64,
    pub created_at: String,
    pub input_type: String,
    pub risk_lane: String,
    pub summary: String,
}

#[derive(Debug, PartialEq, Eq)]
pub struct StoryMatrixRecord {
    pub id: String,
    pub title: String,
    pub status: String,
    pub unit: i64,
    pub integration: i64,
    pub e2e: i64,
    pub platform: i64,
    pub evidence: Option<String>,
}

#[derive(Debug, PartialEq, Eq)]
pub struct StoryVerifyStatus {
    pub id: String,
    pub verify_command: Option<String>,
    pub last_verified_result: Option<String>,
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum GuardrailStatus {
    Active,
    Superseded,
}

impl GuardrailStatus {
    pub fn as_db_value(&self) -> &'static str {
        match self {
            Self::Active => "active",
            Self::Superseded => "superseded",
        }
    }
}

impl FromStr for GuardrailStatus {
    type Err = ParseHarnessValueError;

    fn from_str(value: &str) -> Result<Self, Self::Err> {
        match normalize_token(value).as_str() {
            "active" => Ok(Self::Active),
            "superseded" => Ok(Self::Superseded),
            _ => Err(ParseHarnessValueError::GuardrailStatus(value.to_owned())),
        }
    }
}

#[derive(Debug, PartialEq, Eq)]
pub struct GuardrailRecord {
    pub id: i64,
    pub status: String,
    pub guardrail: String,
    pub rationale: Option<String>,
    pub source: Option<String>,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum GuardrailFilter {
    All,
    Active,
    Superseded,
}

#[derive(Debug, PartialEq, Eq)]
pub struct BacklogRecord {
    pub id: i64,
    pub title: String,
    pub status: String,
    pub risk: Option<String>,
    pub predicted_impact: Option<String>,
    pub actual_outcome: Option<String>,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum BacklogFilter {
    All,
    Open,
    Closed,
}

#[derive(Debug, PartialEq, Eq)]
pub struct DecisionRecord {
    pub id: String,
    pub title: String,
    pub status: String,
    pub last_verified_at: Option<String>,
    pub last_verified_result: Option<String>,
}

#[derive(Debug, PartialEq, Eq)]
pub struct TraceRecord {
    pub id: i64,
    pub created_at: String,
    pub outcome: Option<String>,
    pub task_summary: String,
    pub harness_friction: Option<String>,
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub enum TraceQualityTier {
    Incomplete = 0,
    Minimal = 1,
    Standard = 2,
    Detailed = 3,
}

impl TraceQualityTier {
    pub fn label(self) -> &'static str {
        match self {
            Self::Incomplete => "incomplete",
            Self::Minimal => "minimal",
            Self::Standard => "standard",
            Self::Detailed => "detailed",
        }
    }

    pub fn score(self) -> u8 {
        self as u8
    }
}

#[derive(Debug, PartialEq, Eq)]
pub struct TraceScoreSource {
    pub id: i64,
    pub task_summary: String,
    pub intake_id: Option<i64>,
    pub risk_lane: Option<String>,
    pub agent: Option<String>,
    pub actions_taken: Option<String>,
    pub files_read: Option<String>,
    pub files_changed: Option<String>,
    pub decisions_made: Option<String>,
    pub errors: Option<String>,
    pub outcome: Option<String>,
    pub duration_seconds: Option<i64>,
    pub token_estimate: Option<i64>,
    pub harness_friction: Option<String>,
    pub notes: Option<String>,
}

#[derive(Debug, PartialEq, Eq)]
pub struct TraceScoreResult {
    pub trace_id: i64,
    pub achieved: TraceQualityTier,
    pub risk_lane: Option<String>,
    pub required: Option<TraceQualityTier>,
    pub meets_requirement: bool,
    pub missing_minimal: Vec<String>,
    pub missing_standard: Vec<String>,
    pub missing_detailed: Vec<String>,
}

pub fn required_trace_tier_for_lane(risk_lane: &str) -> Option<TraceQualityTier> {
    match risk_lane {
        "tiny" => Some(TraceQualityTier::Minimal),
        "normal" => Some(TraceQualityTier::Standard),
        "high_risk" => Some(TraceQualityTier::Detailed),
        _ => None,
    }
}

pub fn score_trace(source: TraceScoreSource) -> TraceScoreResult {
    let missing_minimal = missing_minimal_fields(&source);
    let missing_standard = if missing_minimal.is_empty() {
        missing_standard_fields(&source)
    } else {
        Vec::new()
    };
    let missing_detailed = if missing_minimal.is_empty() && missing_standard.is_empty() {
        missing_detailed_fields(&source)
    } else {
        Vec::new()
    };

    let achieved = if !missing_minimal.is_empty() {
        TraceQualityTier::Incomplete
    } else if !missing_standard.is_empty() {
        TraceQualityTier::Minimal
    } else if !missing_detailed.is_empty() {
        TraceQualityTier::Standard
    } else {
        TraceQualityTier::Detailed
    };
    let required = source
        .risk_lane
        .as_deref()
        .and_then(required_trace_tier_for_lane);
    let meets_requirement = required.is_none_or(|tier| achieved >= tier);

    TraceScoreResult {
        trace_id: source.id,
        achieved,
        risk_lane: source.risk_lane,
        required,
        meets_requirement,
        missing_minimal,
        missing_standard,
        missing_detailed,
    }
}

fn missing_minimal_fields(source: &TraceScoreSource) -> Vec<String> {
    let mut missing = Vec::new();
    if source.task_summary.trim().len() < 10 {
        missing.push("task_summary: missing or shorter than 10 characters".to_owned());
    }
    if blank(&source.outcome) {
        missing.push("outcome: null".to_owned());
    }
    missing
}

fn missing_standard_fields(source: &TraceScoreSource) -> Vec<String> {
    let mut missing = Vec::new();
    if blank(&source.agent) {
        missing.push("agent: empty".to_owned());
    }
    if short_json_list(&source.actions_taken) {
        missing.push("actions_taken: empty".to_owned());
    }
    if short_json_list(&source.files_read) {
        missing.push("files_read: empty".to_owned());
    }
    if source.files_changed.is_none() {
        missing.push("files_changed: null".to_owned());
    }
    if source.errors.is_none() && source.harness_friction.is_none() {
        missing.push("errors or harness_friction: both null".to_owned());
    }
    missing
}

fn missing_detailed_fields(source: &TraceScoreSource) -> Vec<String> {
    let mut missing = Vec::new();
    if short_json_list(&source.decisions_made) {
        missing.push("decisions_made: empty".to_owned());
    }
    if source.errors.is_none() {
        missing.push("errors: null".to_owned());
    }
    if source.harness_friction.is_none() {
        missing.push("harness_friction: null".to_owned());
    }
    if source.duration_seconds.is_none() && !notes_explain_missing(&source.notes, "duration") {
        missing.push("duration_seconds: null (no explanation in notes)".to_owned());
    }
    if source.token_estimate.is_none() && !notes_explain_missing(&source.notes, "token") {
        missing.push("token_estimate: null (no explanation in notes)".to_owned());
    }
    missing
}

fn blank(value: &Option<String>) -> bool {
    value.as_deref().map(str::trim).unwrap_or("").is_empty()
}

fn short_json_list(value: &Option<String>) -> bool {
    value.as_deref().map(str::trim).unwrap_or("").len() <= 2
}

fn notes_explain_missing(notes: &Option<String>, field: &str) -> bool {
    let Some(notes) = notes.as_deref() else {
        return false;
    };
    let lower = notes.to_ascii_lowercase();
    lower.contains(field)
        && (lower.contains("unavailable")
            || lower.contains("not available")
            || lower.contains("unknown"))
}

#[derive(Debug, PartialEq, Eq)]
pub struct FrictionRecord {
    pub id: i64,
    pub created_at: String,
    pub risk_lane: Option<String>,
    pub input_type: Option<String>,
    pub task_summary: String,
    pub harness_friction: String,
}

#[derive(Debug, PartialEq, Eq)]
pub struct HarnessStats {
    pub intakes: i64,
    pub stories: i64,
    pub decisions: i64,
    pub backlog_items: i64,
    pub traces: i64,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct CsvList(pub Option<String>);

impl CsvList {
    pub fn from_optional(value: Option<String>) -> Self {
        Self(value.filter(|item| !item.is_empty()))
    }

    pub fn as_json_text(&self) -> Option<String> {
        self.0.as_ref().map(|value| {
            let escaped_items = value
                .split(',')
                .map(|item| format!("\"{}\"", escape_json_string(item.trim())))
                .collect::<Vec<_>>()
                .join(",");
            format!("[{escaped_items}]")
        })
    }

    pub fn as_json_text_or_null_literal(&self) -> String {
        self.as_json_text().unwrap_or_else(|| "null".to_owned())
    }
}

impl fmt::Display for CsvList {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        formatter.write_str(&self.as_json_text_or_null_literal())
    }
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub struct BoolFlag(pub i64);

impl BoolFlag {
    pub fn parse(label: &str, value: &str) -> Result<Self, ParseHarnessValueError> {
        match value {
            "0" => Ok(Self(0)),
            "1" => Ok(Self(1)),
            _ => Err(ParseHarnessValueError::BoolFlag(label.to_owned())),
        }
    }
}

pub fn parse_optional_integer(
    label: &str,
    value: Option<String>,
) -> Result<Option<i64>, ParseHarnessValueError> {
    value
        .map(|inner| {
            inner
                .parse::<i64>()
                .map_err(|_| ParseHarnessValueError::Integer(label.to_owned()))
        })
        .transpose()
}

fn escape_json_string(value: &str) -> String {
    value
        .replace('\\', "\\\\")
        .replace('"', "\\\"")
        .replace('\n', "\\n")
        .replace('\r', "\\r")
        .replace('\t', "\\t")
}

pub fn normalize_token(value: &str) -> String {
    let mut normalized = String::new();
    let mut last_was_separator = false;

    for character in value.trim().chars().flat_map(char::to_lowercase) {
        if character.is_ascii_alphanumeric() {
            normalized.push(character);
            last_was_separator = false;
        } else if !last_was_separator && !normalized.is_empty() {
            normalized.push('_');
            last_was_separator = true;
        }
    }

    while normalized.ends_with('_') {
        normalized.pop();
    }

    normalized
}

pub fn yes_no(value: i64) -> String {
    if value == 1 {
        "yes".to_owned()
    } else {
        "no".to_owned()
    }
}

pub fn proof_display(value: i64, numeric: bool) -> String {
    if numeric {
        value.to_string()
    } else {
        yes_no(value)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_input_type_aliases() {
        assert_eq!("new_spec".parse::<InputType>().unwrap(), InputType::NewSpec);
        assert_eq!(
            "maintenance request".parse::<InputType>().unwrap(),
            InputType::Maintenance
        );
        assert_eq!(
            "Harness improvement".parse::<InputType>().unwrap(),
            InputType::HarnessImprovement
        );
    }

    #[test]
    fn parses_high_risk_lane_alias() {
        assert_eq!("high-risk".parse::<RiskLane>().unwrap(), RiskLane::HighRisk);
    }

    #[test]
    fn renders_csv_as_json_text() {
        assert_eq!(
            CsvList::from_optional(Some("auth, data model".to_owned()))
                .as_json_text_or_null_literal(),
            "[\"auth\",\"data model\"]"
        );
        assert_eq!(
            CsvList::from_optional(None).as_json_text_or_null_literal(),
            "null"
        );
    }

    #[test]
    fn parses_bool_flags() {
        assert_eq!(BoolFlag::parse("--unit", "1").unwrap(), BoolFlag(1));
        assert!(BoolFlag::parse("--unit", "yes").is_err());
    }

    fn trace_source() -> TraceScoreSource {
        TraceScoreSource {
            id: 7,
            task_summary: "Completed a useful task".to_owned(),
            intake_id: None,
            risk_lane: None,
            agent: None,
            actions_taken: None,
            files_read: None,
            files_changed: None,
            decisions_made: None,
            errors: None,
            outcome: Some("completed".to_owned()),
            duration_seconds: None,
            token_estimate: None,
            harness_friction: None,
            notes: None,
        }
    }

    #[test]
    fn scores_minimal_standard_and_detailed_traces() {
        let minimal = score_trace(trace_source());
        assert_eq!(minimal.achieved, TraceQualityTier::Minimal);

        let mut standard_source = trace_source();
        standard_source.agent = Some("codex".to_owned());
        standard_source.actions_taken = Some("[\"read\",\"patched\"]".to_owned());
        standard_source.files_read = Some("[\"PHASE3.md\"]".to_owned());
        standard_source.files_changed = Some("[\"docs/TRACE_SPEC.md\"]".to_owned());
        standard_source.harness_friction = Some("none".to_owned());
        let standard = score_trace(standard_source);
        assert_eq!(standard.achieved, TraceQualityTier::Standard);

        let mut detailed_source = trace_source();
        detailed_source.agent = Some("codex".to_owned());
        detailed_source.actions_taken = Some("[\"read\",\"patched\"]".to_owned());
        detailed_source.files_read = Some("[\"PHASE3.md\"]".to_owned());
        detailed_source.files_changed = Some("[\"docs/TRACE_SPEC.md\"]".to_owned());
        detailed_source.decisions_made = Some("[\"kept schema unchanged\"]".to_owned());
        detailed_source.errors = Some("[\"none\"]".to_owned());
        detailed_source.harness_friction = Some("none".to_owned());
        detailed_source.duration_seconds = Some(120);
        detailed_source.token_estimate = Some(2000);
        let detailed = score_trace(detailed_source);
        assert_eq!(detailed.achieved, TraceQualityTier::Detailed);
    }

    #[test]
    fn compares_trace_score_to_lane_requirement() {
        let mut source = trace_source();
        source.risk_lane = Some("high_risk".to_owned());
        source.agent = Some("codex".to_owned());
        source.actions_taken = Some("[\"read\",\"patched\"]".to_owned());
        source.files_read = Some("[\"PHASE3.md\"]".to_owned());
        source.files_changed = Some("[\"docs/TRACE_SPEC.md\"]".to_owned());
        source.harness_friction = Some("none".to_owned());

        let result = score_trace(source);

        assert_eq!(result.achieved, TraceQualityTier::Standard);
        assert_eq!(result.required, Some(TraceQualityTier::Detailed));
        assert!(!result.meets_requirement);
        assert!(result
            .missing_detailed
            .iter()
            .any(|field| field.starts_with("decisions_made")));
    }
}
