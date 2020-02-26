package attestations

import (
	"context"
	"flag"
	"path"
	"reflect"
	"testing"

	ethpb "github.com/prysmaticlabs/ethereumapis/eth/v1alpha1"
	"github.com/prysmaticlabs/prysm/shared/cmd"
	"github.com/prysmaticlabs/prysm/slasher/db"
	"github.com/prysmaticlabs/prysm/slasher/db/kv"
	"github.com/prysmaticlabs/prysm/slasher/flags"
	"github.com/urfave/cli"
)

const slasherDBName = "slasherdata"

func TestSpanDetector_DetectSlashingForValidator(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	ctx := cli.NewContext(app, set, nil)
	baseDir := ctx.GlobalString(cmd.DataDirFlag.Name)
	dbPath := path.Join(baseDir, slasherDBName)
	cfg := &kv.Config{SpanCacheEnabled: ctx.GlobalBool(flags.UseSpanCacheFlag.Name)}
	d, err := db.NewDB(dbPath, cfg)
	context := context.Background()
	if err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}

	type testStruct struct {
		name                     string
		sourceEpoch              uint64
		targetEpoch              uint64
		slashableEpoch           uint64
		shouldSlash              bool
		spansByEpochForValidator map[uint64][2]uint16
	}
	tests := []testStruct{
		{
			name:           "Should slash if max span > distance",
			sourceEpoch:    3,
			targetEpoch:    6,
			slashableEpoch: 7,
			shouldSlash:    true,
			// Given a distance of (6 - 3) = 3, we want the validator at epoch 3 to have
			// committed a slashable offense by having a max span of 4 > distance.
			spansByEpochForValidator: map[uint64][2]uint16{
				3: {0, 4},
			},
		},
		{
			name:        "Should NOT slash if max span < distance",
			sourceEpoch: 3,
			targetEpoch: 6,
			// Given a distance of (6 - 3) = 3, we want the validator at epoch 3 to NOT
			// have committed slashable offense by having a max span of 1 < distance.
			shouldSlash: false,
			spansByEpochForValidator: map[uint64][2]uint16{
				3: {0, 1},
			},
		},
		{
			name:        "Should NOT slash if max span == distance",
			sourceEpoch: 3,
			targetEpoch: 6,
			// Given a distance of (6 - 3) = 3, we want the validator at epoch 3 to NOT
			// have committed slashable offense by having a max span of 3 == distance.
			shouldSlash: false,
			spansByEpochForValidator: map[uint64][2]uint16{
				3: {0, 3},
			},
		},
		{
			name:        "Should NOT slash if min span == 0",
			sourceEpoch: 3,
			targetEpoch: 6,
			// Given a min span of 0 and no max span slashing, we want validator to NOT
			// have committed a slashable offense if min span == 0.
			shouldSlash: false,
			spansByEpochForValidator: map[uint64][2]uint16{
				3: {0, 1},
			},
		},
		{
			name:        "Should slash if min span > 0 and min span < distance",
			sourceEpoch: 3,
			targetEpoch: 6,
			// Given a distance of (6 - 3) = 3, we want the validator at epoch 3 to have
			// committed a slashable offense by having a min span of 1 < distance.
			shouldSlash:    true,
			slashableEpoch: 4,
			spansByEpochForValidator: map[uint64][2]uint16{
				3: {1, 0},
			},
		},
		// Proto Max Span Tests from the eth2-surround repo.
		{
			name:        "Proto max span test #1",
			sourceEpoch: 8,
			targetEpoch: 18,
			shouldSlash: false,
			spansByEpochForValidator: map[uint64][2]uint16{
				0: {4, 0},
				1: {2, 0},
				2: {1, 0},
				4: {0, 2},
				5: {0, 1},
			},
		},
		{
			name:           "Proto max span test #2",
			sourceEpoch:    4,
			targetEpoch:    12,
			shouldSlash:    false,
			slashableEpoch: 0,
			spansByEpochForValidator: map[uint64][2]uint16{
				4:  {14, 2},
				5:  {13, 1},
				6:  {12, 0},
				7:  {11, 0},
				9:  {0, 9},
				10: {0, 8},
				11: {0, 7},
				12: {0, 6},
				13: {0, 5},
				14: {0, 4},
				15: {0, 3},
				16: {0, 2},
				17: {0, 1},
			},
		},
		{
			name:           "Proto max span test #3",
			sourceEpoch:    10,
			targetEpoch:    15,
			shouldSlash:    true,
			slashableEpoch: 18,
			spansByEpochForValidator: map[uint64][2]uint16{
				4:  {14, 2},
				5:  {13, 7},
				6:  {12, 6},
				7:  {11, 5},
				8:  {0, 4},
				9:  {0, 9},
				10: {0, 8},
				11: {0, 7},
				12: {0, 6},
				13: {0, 5},
				14: {0, 4},
				15: {0, 3},
				16: {0, 2},
				17: {0, 1},
			},
		},
		// Proto Min Span Tests from the eth2-surround repo.
		{
			name:        "Proto min span test #1",
			sourceEpoch: 4,
			targetEpoch: 6,
			shouldSlash: false,
			spansByEpochForValidator: map[uint64][2]uint16{
				1: {5, 0},
				2: {4, 0},
				3: {3, 0},
			},
		},
		{
			name:        "Proto min span test #2",
			sourceEpoch: 11,
			targetEpoch: 15,
			shouldSlash: false,
			spansByEpochForValidator: map[uint64][2]uint16{
				1:  {5, 0},
				2:  {4, 0},
				3:  {3, 0},
				4:  {14, 0},
				5:  {13, 1},
				6:  {12, 0},
				7:  {11, 0},
				8:  {10, 0},
				9:  {9, 0},
				10: {8, 0},
				11: {7, 0},
				12: {6, 0},
				14: {0, 4},
				15: {0, 3},
				16: {0, 2},
				17: {0, 1},
			},
		},
		{
			name:           "Proto min span test #3",
			sourceEpoch:    9,
			targetEpoch:    19,
			shouldSlash:    true,
			slashableEpoch: 14,
			spansByEpochForValidator: map[uint64][2]uint16{
				0:  {5, 0},
				1:  {4, 0},
				2:  {3, 0},
				3:  {11, 0},
				4:  {10, 1},
				5:  {9, 0},
				6:  {8, 0},
				7:  {7, 0},
				8:  {6, 0},
				9:  {5, 0},
				10: {7, 0},
				11: {6, 3},
				12: {0, 2},
				13: {0, 1},
				14: {0, 3},
				15: {0, 2},
				16: {0, 1},
				17: {0, 0},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := &SpanDetector{
				db: d,
			}
			// We only care about validator index 0 for these tests for simplicity.
			validatorIndex := uint64(0)
			for k, v := range tt.spansByEpochForValidator {
				err := sd.db.SaveValidatorEpochSpans(context, validatorIndex, k, v)
				if err != nil {
					t.Fatalf("Failed to write to db: %v", err)
				}
			}
			attData := &ethpb.AttestationData{
				Source: &ethpb.Checkpoint{
					Epoch: tt.sourceEpoch,
				},
				Target: &ethpb.Checkpoint{
					Epoch: tt.targetEpoch,
				},
			}
			res, err := sd.DetectSlashingForValidator(context, validatorIndex, attData)
			if err != nil {
				t.Fatal(err)
			}
			if !tt.shouldSlash && res != nil {
				t.Fatalf("Did not want validator to be slashed but found slashable offense: %v", res)
			}
			if tt.shouldSlash {
				want := &DetectionResult{
					Kind:           SurroundVote,
					SlashableEpoch: tt.slashableEpoch,
				}
				if !reflect.DeepEqual(res, want) {
					t.Errorf("Wanted: %v, received %v", want, res)
				}
			}
		})
	}
}

func TestSpanDetector_DetectSlashingForValidator_MultipleValidators(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	ctx := cli.NewContext(app, set, nil)
	baseDir := ctx.GlobalString(cmd.DataDirFlag.Name)
	dbPath := path.Join(baseDir, slasherDBName)
	cfg := &kv.Config{SpanCacheEnabled: ctx.GlobalBool(flags.UseSpanCacheFlag.Name)}
	d, err := db.NewDB(dbPath, cfg)
	if err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}
	context := context.Background()
	type testStruct struct {
		name            string
		sourceEpochs    []uint64
		targetEpochs    []uint64
		slashableEpochs []uint64
		shouldSlash     []bool
		spansByEpoch    []map[uint64][2]uint16
	}
	tests := []testStruct{
		{
			name:            "3 of 5 validators slashed",
			sourceEpochs:    []uint64{0, 2, 4, 5, 1},
			targetEpochs:    []uint64{10, 3, 5, 9, 8},
			slashableEpochs: []uint64{6, 0, 7, 8, 0},
			// Detections - surrounding, none, surrounded, surrounding, none.
			shouldSlash: []bool{true, false, true, true, false},
			// Atts in map: (src, epoch) - 0: (2, 6), 1: (1, 2), 2: (1, 7), 3: (6, 8), 4: (0, 3)
			spansByEpoch: []map[uint64][2]uint16{
				// Epoch 0.
				{
					0: {6, 0},
					1: {2, 0},
					2: {7, 0},
					3: {8, 0},
				},
				// Epoch 1.
				{
					0: {5, 0},
					3: {7, 0},
					4: {0, 1},
				},
				// Epoch 2.
				{
					2: {0, 5},
					3: {6, 0},
					4: {0, 2},
				},
				// Epoch 3.
				{
					0: {0, 3},
					2: {0, 4},
					3: {5, 0},
				},
				// Epoch 4.
				{
					0: {0, 2},
					2: {0, 3},
					3: {4, 0},
				},
				// Epoch 5.
				{
					0: {0, 1},
					2: {0, 2},
					3: {3, 0},
				},
				// Epoch 6.
				{
					2: {0, 1},
				},
				// Epoch 7.
				{
					3: {0, 1},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sd := &SpanDetector{
				db: d,
			}
			for i := uint64(0); i < uint64(len(tt.spansByEpoch)); i++ {
				sd.db.SaveEpochSpansMap(context, i, tt.spansByEpoch[i])
			}
			for valIdx := uint64(0); valIdx < uint64(len(tt.shouldSlash)); valIdx++ {
				attData := &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{
						Epoch: tt.sourceEpochs[valIdx],
					},
					Target: &ethpb.Checkpoint{
						Epoch: tt.targetEpochs[valIdx],
					},
				}
				res, err := sd.DetectSlashingForValidator(context, valIdx, attData)
				if err != nil {
					t.Fatal(err)
				}
				if !tt.shouldSlash[valIdx] && res != nil {
					t.Fatalf("Did not want validator to be slashed but found slashable offense: %v", res)
				}
				if tt.shouldSlash[valIdx] {
					want := &DetectionResult{
						Kind:           SurroundVote,
						SlashableEpoch: tt.slashableEpochs[valIdx],
					}
					if !reflect.DeepEqual(res, want) {
						t.Errorf("Wanted: %v, received %v", want, res)
					}
				}
			}
		})
	}
}

func TestNewSpanDetector_UpdateSpans(t *testing.T) {
	app := cli.NewApp()
	set := flag.NewFlagSet("test", 0)
	ctx := cli.NewContext(app, set, nil)
	baseDir := ctx.GlobalString(cmd.DataDirFlag.Name)
	dbPath := path.Join(baseDir, slasherDBName)
	cfg := &kv.Config{SpanCacheEnabled: ctx.GlobalBool(flags.UseSpanCacheFlag.Name)}
	d, err := db.NewDB(dbPath, cfg)
	if err != nil {
		t.Fatalf("Failed to init db: %v", err)
	}
	context := context.Background()
	sd := &SpanDetector{
		db: d,
	}
	type testStruct struct {
		name      string
		att       *ethpb.IndexedAttestation
		numEpochs uint64
		want      []map[uint64][2]uint16
	}
	tests := []testStruct{
		{
			name: "Distance of 2 should update max spans accordingly",
			att: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0, 1, 2},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{
						Epoch: 1,
					},
					Target: &ethpb.Checkpoint{
						Epoch: 3,
					},
				},
			},
			numEpochs: 3,
			want: []map[uint64][2]uint16{
				// Epoch 0.
				{
					0: {3, 0},
					1: {3, 0},
					2: {3, 0},
				},
				// Epoch 1.
				{},
				// Epoch 2.
				{
					0: {0, 1},
					1: {0, 1},
					2: {0, 1},
				},
			},
		},
		{
			name: "Distance of 4 should update max spans accordingly",
			att: &ethpb.IndexedAttestation{
				AttestingIndices: []uint64{0, 1, 2},
				Data: &ethpb.AttestationData{
					Source: &ethpb.Checkpoint{
						Epoch: 0,
					},
					Target: &ethpb.Checkpoint{
						Epoch: 5,
					},
				},
			},
			numEpochs: 5,
			want: []map[uint64][2]uint16{
				// Epoch 0.
				{},
				// Epoch 1.
				{
					0: {0, 4},
					1: {0, 4},
					2: {0, 4},
				},
				// Epoch 2.
				{
					0: {0, 3},
					1: {0, 3},
					2: {0, 3},
				},
				// Epoch 3.
				{
					0: {0, 2},
					1: {0, 2},
					2: {0, 2},
				},
				// Epoch 4.
				{
					0: {0, 1},
					1: {0, 1},
					2: {0, 1},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := sd.UpdateSpans(context, tt.att); err != nil {
				t.Fatal(err)
			}
			for epoch := 0; epoch < len(tt.want); epoch++ {
				sm, err := sd.db.EpochSpansMap(context, uint64(epoch))
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(sm, tt.want[epoch]) {
					t.Errorf("Wanted spans %v, received %v", tt.want[epoch], sm)
				}
				sd.db.DeleteEpochSpans(context, uint64(epoch))
			}

		})
	}
}