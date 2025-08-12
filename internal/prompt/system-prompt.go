package prompt

var SystemPromt = `
You are a data architect who extracts structured data from text.
From the given text, extract all entities and the relationships between them, paying close attention to the reasons and motivations behind events.

**Desired Output Format:**
Your output MUST be a single, valid JSON object with two keys: "entities" and "relations".

**Entities Guideline:**
- "entities" should be an array of objects. Each object must have "ID", "Name", "Label", and "Properties".
- For entities of "Event" label, if the text describes a reason for the event, add a "reason" key to its "Properties".

**Relations Guideline:**
- "relations" should be an array of objects. Each object must have "SourceName", "TargetName", and "Type".
- Use descriptive "Type"s. For factual connections, use types like 'PLAYS_FOR' or 'WON'.
- **For causal or motivational connections, use abstract types like 'MOTIVATED_BY', 'INFLUENCED_BY', or 'REASON_FOR'.**

**Identifier Guideline:**
- The "ID" for entities and the "SourceName"/"TargetName" for relations should be a consistent, snake_case identifier.

**Text to process:**
2025년 8월 3일 토트넘 핫스퍼의 축구선수인 손흥민은 팀을 떠나기로 결정했다. 그동안 178골 107 어시스트를 기록한 이 한국인 선수는 대한민국 국가대표 주장이자 토트넘 핫스퍼의 주장이다. 그는 2015년 처음 토트넘 핫스퍼에 이적하였고 마지막 시즌인 24/25시즌에 유로파 대회를 우승하여 팀에게 17년만에 우승컵을 안겨주었다. 그는 미국 1부 리그인 MLS의 서부리그인 LA FC로 이적을 하기로 결정하였고, 2025년 8월 8일 입단을 완료하였다. 그가 LA FC로 이적을 결정한 이유는 다음과 같다.
1. 2026년 월드컵은 미국에서 열린다. 이번 월드컵을 선수 생활 중 마지막으로 참가한다고 생각한 손흥민은 최상의 결과를 위해 미리 미국으로 이적했다고 밝혔다.
2. 많은 팀 중 LA FC의 회장이 직접 전화를 걸어 포부와 미래 그리고 기대와 처우에 대해서 감명깊게 대화한 것이 이적의 주요 포인트였다고 한다.
`

/*
* 받은 결과
{
  "entities": [
    {
      "ID": "son_heung_min",
      "Name": "손흥민",
      "Label": "Player",
      "Embedding": [],
      "Properties": {
        "Nationality": "대한민국",
        "Position": "축구선수",
        "Goals": 178,
        "Assists": 107
      }
    },
    {
      "ID": "tottenham_hotspur",
      "Name": "토트넘 핫스퍼",
      "Label": "Team",
      "Embedding": [],
      "Properties": {
        "League": null
      }
    },
    {
      "ID": "2025_08_03_departure_event",
      "Name": "손흥민 토트넘 핫스퍼 이적",
      "Label": "Event",
      "Embedding": [],
      "Properties": {
        "Date": "2025-08-03",
        "EventType": "선수 이적",
        "Description": "손흥민이 토트넘 핫스퍼를 떠나기로 결정함"
      }
    },
    {
      "ID": "2015_tottenham_transfer_event",
      "Name": "손흥민 토트넘 핫스퍼 합류",
      "Label": "Event",
      "Embedding": [],
      "Properties": {
        "Year": 2015,
        "EventType": "선수 이적",
        "Description": "손흥민이 토트넘 핫스퍼에 이적"
      }
    },
    {
      "ID": "24_25_europa_league_win_event",
      "Name": "24/25시즌 유로파리그 우승",
      "Label": "Event",
      "Embedding": [],
      "Properties": {
        "Season": "24/25",
        "EventType": "우승",
        "Competition": "유로파 리그",
        "Description": "토트넘 핫스퍼가 24/25시즌 유로파 리그 우승"
      }
    },
    {
      "ID": "korea_national_team",
      "Name": "대한민국 국가대표",
      "Label": "Team",
      "Embedding": [],
      "Properties": {
        "Type": "국가대표"
      }
    }
  ],
  "relations": [
    {
      "SourceName": "son_heung_min",
      "TargetName": "tottenham_hotspur",
      "Type": "플레이어_소속"
    },
    {
      "SourceName": "2025_08_03_departure_event",
      "TargetName": "son_heung_min",
      "Type": "이적_대상"
    },
     {
      "SourceName": "2025_08_03_departure_event",
      "TargetName": "tottenham_hotspur",
      "Type": "이적_팀"
    },
    {
      "SourceName": "2015_tottenham_transfer_event",
      "TargetName": "son_heung_min",
      "Type": "합류_선수"
    },
    {
      "SourceName": "2015_tottenham_transfer_event",
      "TargetName": "tottenham_hotspur",
      "Type": "합류_팀"
    },
    {
      "SourceName": "24_25_europa_league_win_event",
      "TargetName": "tottenham_hotspur",
      "Type": "우승_팀"
    },
     {
      "SourceName": "son_heung_min",
      "TargetName": "korea_national_team",
      "Type": "플레이어_소속"
    }
  ]
}
*/

const EvaluatePromptTemplate = `
You are a highly intelligent graph evaluator for a Retrieval-Augmented Generation system.
Your task is to evaluate the usefulness of a given Subgraph for answering a specific User Query.

Based on the following criteria, please provide a score from 0.0 to 1.0.
1.  **Richness**: How much useful information does the subgraph contain?
2.  **Relevance**: How directly relevant is the information to the User Query? Ignore irrelevant service.
3.  **Connectivity**: Are the entities and relationships well-connected to form a coherent story for the query?

Provide your output ONLY in JSON format like this: {"score": 0.85, "reason": "The subgraph is highly relevant..."}

---
**User Query:** "%s"

**Subgraph to Evaluate:**
%s
---
`

const EntityExtractionPromptTemplate = `
You are a Named Entity Recognition specialist.
From the User Query below, extract all key entities such as people, organizations, locations, or concepts.
Extract ONLY the names of the entities.

Your output MUST be a JSON array of strings, like this: ["entity1", "entity2", "entity3"]

---
**User Query:** "%s"
---
`

const FinalPromptTemplate = `
You are a helpful AI assistant answering questions based on the context provided from a knowledge graph.
Your task is to synthesize the information in the 'Context' section to answer the 'User's Question'.
Answer ONLY with the information provided in the context. Do not use any of your prior knowledge.
If the context does not contain the answer, say that you cannot find the answer in the provided information.
Answer in Korean.

---
**[Context from Knowledge Graph]**
%s
---
**[User's Question]**
%s
`