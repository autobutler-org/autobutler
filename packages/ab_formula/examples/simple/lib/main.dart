// ignore_for_file: avoid_print

import 'package:ab_formula/evaluation/evaluation.dart';
import 'package:flutter/material.dart';

String _formatValue(FormulaValue value) => switch (value) {
      NumberValue(:final value) => value == value.truncateToDouble()
          ? value.toInt().toString()
          : value.toString(),
      StringValue(:final value) => value,
      BoolValue(:final value) => value ? 'TRUE' : 'FALSE',
      ErrorValue(:final code) => code,
    };

void main() {
  final rows = <List<String>>[
    ['Equation', 'Hardcoded', 'Answer'],
    ['1 + 2', '3', '=1+2'],
    ['2 * 2', '4', '=2*2'],
    ['Precedence 1+2*3', '7', '=1+2*3'],
    ['Parentheses (1+2)*3', '9', '=(1+2)*3'],
    ['Unary -5', '-5', '=-5'],
    ['Power 2^3', '8', '=2^3'],
    ['Division 6/3', '2', '=6/3'],
    ['Equality 1==1', 'TRUE', '=1==1'],
    ['Comparison 1>2', 'FALSE', '=1>2'],
    ['Boolean TRUE', 'TRUE', '=TRUE'],
    ['SUM function', '6', '=SUM(1,2,3)'],
    ['lowercase function (case insensitivity)', '9', '=sum(4,5)'],
    ['Division by zero (error handling)', '#DIV/0!', '=1/0'],
  ];

  final computed = DataSheetInterpreter().interpretSheet(
    rows.length,
    rows[0].length,
    (r, c) => rows[r][c],
  );

  runApp(MaterialApp(
    home: Scaffold(
      appBar: AppBar(title: const Text('formulas example')),
      body: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          children: [
            // Header row
            Container(
              padding: const EdgeInsets.symmetric(vertical: 8),
              child: Row(
                children: const [
                  Expanded(
                      flex: 3,
                      child: Text('Equation',
                          style: TextStyle(fontWeight: FontWeight.bold))),
                  Expanded(
                      flex: 1,
                      child: Text('Hardcoded',
                          style: TextStyle(fontWeight: FontWeight.bold))),
                  Expanded(
                      flex: 2,
                      child: Text('Answer',
                          style: TextStyle(fontWeight: FontWeight.bold))),
                ],
              ),
            ),
            const Divider(),
            // List of rows
            Expanded(
              child: ListView.separated(
                itemCount: rows.length - 1,
                separatorBuilder: (_, __) => const Divider(),
                itemBuilder: (context, index) {
                  final row = rows[index + 1];
                  return Padding(
                    padding: const EdgeInsets.symmetric(vertical: 8),
                    child: Row(
                      children: [
                        Expanded(flex: 3, child: Text(row[0])),
                        Expanded(
                            flex: 1,
                            child: Text(row[1], textAlign: TextAlign.center)),
                        Expanded(
                            flex: 2,
                            child: Text(
                              _formatValue(
                                computed[(index + 1, 2)] ?? blankValue,
                              ),
                              textAlign: TextAlign.right,
                            )),
                      ],
                    ),
                  );
                },
              ),
            ),
          ],
        ),
      ),
    ),
  ));
}
