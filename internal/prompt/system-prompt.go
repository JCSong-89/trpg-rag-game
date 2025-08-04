package prompt

var SystemPromt = `
다음의 내용을 다음의 포맷형식에 대응 될 수 있도록 파싱해줘. 
언어는 Go언어이며 각각 쿼드란트 와 그래프 DB인 Neo4j에 들어갈 것이고 
중요도는 입력 후 나중에 한번에 계산할 것이다.  
아래는 각 데이터들의 객체야.
// EntityData는 애플리케이션에서 처리할 엔티티 정보를 담습니다.
type EntityData struct {
	ID          string
	Name        string
	Label       string    // Neo4j 노드 레이블 (예: Warrior, Weapon, Event)
	Embedding   []float64 // 벡터 임베딩 (시뮬레이션)
	Properties  map[string]any // 이벤트 노드를 위한 추가 속성
}

// RelationData는 엔티티 간의 관계를 정의합니다.
type RelationData struct {
	SourceName string
	TargetName string
	Type       string
}  추가로  
엔티티 데이터에 이벤트 관련 엔티티도 추가해줘.
 
주는 데이터는 다음과 같다. 
1. 쿼드란트에 삽입할 데이터  
2. Neo4j에 삽입할 데이터  
3. 이벤트, 연관관계, 명사등을 확실히 구분하여 만들 것 
4. Go언어 형식에 대응하도록 만들 것  
5. 위 제시한 객체 타입을 중심으로 만들 것   
6. JSON형식으로 줄것
7. 코드 작성을 원하는 것이 아닌 데이터 파싱임을 잊지 말것

<대상 내용> 
2025년 8월 3일 토트넘 핫스퍼의 축구선수인 손흥민은 팀을 떠나기로 결정했다. 그동안 178골 107 어시스트를 기록한 이 한국인 선수는 대한민국 국가대표 주장이자 토트넘 핫스퍼의 주장이다. 그는 2015년 처음 토트넘 핫스퍼에 이적하였고 마지막 시즌인 24/25시즌에 유로파 대회를 우승하여 팀에게 17년만에 우승컵을 안겨주었다. 
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