import { describe, expect, it } from 'vitest';
import { colName, readXlsx, writeXlsx } from './xlsx';

describe('colName', () => {
	it('names columns like a spreadsheet', () => {
		expect([0, 25, 26, 27, 701, 702].map(colName)).toEqual(['A', 'Z', 'AA', 'AB', 'ZZ', 'AAA']);
	});
});

describe('xlsx round trip', () => {
	it('reads back what it writes, including XML-special and unicode text', async () => {
		const rows = [
			['Title', 'Points', 'Note'],
			['A & B <c>', '3', 'Çağrı "quoted"'],
			['', '1.5', 'line1\nline2']
		];
		const bytes = writeXlsx(rows, 'Items');
		expect(bytes[0]).toBe(0x50); // PK
		expect(await readXlsx(bytes)).toEqual(rows);
	});
	it('reads deflated files with shared strings', async () => {
		// A tiny workbook in the layout Excel writes (shared strings, numbers), built and deflated here.
		const files: Record<string, string> = {
			'[Content_Types].xml': '<Types/>',
			'xl/workbook.xml': '<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="r"><sheets><sheet name="S" sheetId="1" r:id="rId1"/></sheets></workbook>',
			'xl/_rels/workbook.xml.rels': '<Relationships><Relationship Id="rId1" Target="worksheets/sheet1.xml"/></Relationships>',
			'xl/sharedStrings.xml': '<sst><si><t>Name</t></si><si><r><t>Ri</t></r><r><t>ch</t></r></si></sst>',
			'xl/worksheets/sheet1.xml':
				'<worksheet><sheetData><row r="1"><c r="A1" t="s"><v>0</v></c><c r="C1"><v>42</v></c></row><row r="3"><c r="B3" t="s"><v>1</v></c><c r="C3" t="b"><v>1</v></c></row></sheetData></worksheet>'
		};
		const { zipDeflated } = await import('./xlsx');
		const bytes = await zipDeflated(files);
		expect(await readXlsx(bytes)).toEqual([
			['Name', '', '42'],
			['', '', ''],
			['', 'Rich', 'TRUE']
		]);
	});
});
